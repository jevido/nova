// Package nova is a small client for the Nova (pixeldrain-compatible) storage API.
// It has no Wails dependencies so it can be tested as plain Go.
package nova

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultBaseURL = "https://nova.storage"

// ErrUnauthorized is returned when the API rejects the credentials.
var ErrUnauthorized = errors.New("not authorized, please sign in again")

// APIError is a non-2xx response from the API.
type APIError struct {
	Status  int        `json:"status"`
	Value   string     `json:"value"`
	Message string     `json:"message"`
	Errors  []APIError `json:"errors"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		msgs := make([]string, len(e.Errors))
		for i := range e.Errors {
			msgs[i] = e.Errors[i].Error()
		}
		return strings.Join(msgs, "; ")
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Value != "" {
		return strings.ReplaceAll(e.Value, "_", " ")
	}
	return fmt.Sprintf("request failed with status %d", e.Status)
}

// Client talks to a Nova server. It is safe for concurrent use.
type Client struct {
	mu      sync.RWMutex
	baseURL string
	apiKey  string
	http    *http.Client
	// transfer client has no overall timeout so large transfers are not cut off.
	transfer *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	c := &Client{
		http:     &http.Client{Timeout: 60 * time.Second},
		transfer: &http.Client{},
	}
	c.SetBaseURL(baseURL)
	c.SetAPIKey(apiKey)
	return c
}

func (c *Client) SetBaseURL(u string) {
	u = strings.TrimRight(strings.TrimSpace(u), "/")
	if u == "" {
		u = DefaultBaseURL
	}
	c.mu.Lock()
	c.baseURL = u
	c.mu.Unlock()
}

func (c *Client) BaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

func (c *Client) SetAPIKey(k string) {
	c.mu.Lock()
	c.apiKey = strings.TrimSpace(k)
	c.mu.Unlock()
}

func (c *Client) APIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey
}

// EncodePath escapes each segment of a filesystem path.
func EncodePath(p string) string {
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		if s == "" {
			continue
		}
		out = append(out, url.PathEscape(s))
	}
	return "/" + strings.Join(out, "/")
}

// FileURL is the API URL of a filesystem node, with an optional raw query.
func (c *Client) FileURL(p, rawQuery string) string {
	u := c.BaseURL() + "/api/filesystem" + EncodePath(p)
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	return u
}

func (c *Client) newRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	if k := c.APIKey(); k != "" {
		req.SetBasicAuth("", k)
	}
	req.Header.Set("User-Agent", "nova-desktop/0.1")
	return req, nil
}

// Do performs a request against an absolute URL using the transfer client and
// returns the raw response. The caller must close the body. Errors are decoded.
func (c *Client) Do(ctx context.Context, method, rawURL string, body io.Reader, hdr http.Header) (*http.Response, error) {
	req, err := c.newRequest(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range hdr {
		req.Header[k] = v
	}
	res, err := c.transfer.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		defer res.Body.Close()
		return nil, decodeError(res)
	}
	return res, nil
}

func decodeError(res *http.Response) error {
	if res.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	apiErr := &APIError{Status: res.StatusCode}
	b, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	_ = json.Unmarshal(b, apiErr)
	if apiErr.Message == "" && apiErr.Value == "" && res.StatusCode == http.StatusNotFound {
		apiErr.Message = "not found"
	}
	return apiErr
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, form url.Values, out any) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	u := endpoint
	if !strings.HasPrefix(u, "http") {
		u = c.BaseURL() + "/api/" + strings.TrimLeft(endpoint, "/")
	}
	req, err := c.newRequest(ctx, method, u, body)
	if err != nil {
		return err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return decodeError(res)
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// ---- Users ----

type Subscription struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	FileSizeLimit      int64  `json:"file_size_limit"`
	StorageLimit       int64  `json:"storage_limit"`
	MonthlyTransferCap int64  `json:"monthly_transfer_cap"`
}

type User struct {
	ID                    string       `json:"id"`
	Username              string       `json:"username"`
	Email                 string       `json:"email"`
	Subscription          Subscription `json:"subscription"`
	StorageSpaceUsed      int64        `json:"storage_space_used"`
	FilesystemStorageUsed int64        `json:"filesystem_storage_used"`
	FilesystemNodeCount   int64        `json:"filesystem_node_count"`
	MonthlyTransferCap    int64        `json:"monthly_transfer_cap"`
	MonthlyTransferUsed   int64        `json:"monthly_transfer_used"`
	BalanceMicroEUR       int64        `json:"balance_micro_eur"`
}

func (c *Client) User(ctx context.Context) (*User, error) {
	var u User
	if err := c.doJSON(ctx, http.MethodGet, "user", nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// Login exchanges a username and password for an API key.
func (c *Client) Login(ctx context.Context, username, password, appName string) (string, error) {
	var out struct {
		AuthKey string `json:"auth_key"`
	}
	form := url.Values{"username": {username}, "password": {password}, "app_name": {appName}}
	if err := c.doJSON(ctx, http.MethodPost, "user/login", form, &out); err != nil {
		return "", err
	}
	if out.AuthKey == "" {
		return "", errors.New("login failed")
	}
	return out.AuthKey, nil
}

// Logout invalidates the current session key.
func (c *Client) Logout(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodDelete, "user/session", nil, nil)
}

// ---- Filesystem ----

type Permissions struct {
	Owner  bool `json:"owner"`
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
}

type Node struct {
	ID              string       `json:"id"`
	Type            string       `json:"type"`
	Path            string       `json:"path"`
	Name            string       `json:"name"`
	Created         time.Time    `json:"created"`
	Modified        time.Time    `json:"modified"`
	ModeString      string       `json:"mode_string"`
	ModeOctal       string       `json:"mode_octal"`
	CreatedBy       string       `json:"created_by"`
	FileSize        int64        `json:"file_size"`
	FileType        string       `json:"file_type"`
	SHA256          string       `json:"sha256_sum"`
	LinkPermissions *Permissions `json:"link_permissions,omitempty"`
}

func (n Node) IsDir() bool { return n.Type == "dir" }

type Listing struct {
	Path        []Node      `json:"path"`
	BaseIndex   int         `json:"base_index"`
	Children    []Node      `json:"children"`
	Permissions Permissions `json:"permissions"`
}

// Stat returns the node at p and, for directories, its children.
func (c *Client) Stat(ctx context.Context, p string) (*Listing, error) {
	var l Listing
	if err := c.doJSON(ctx, http.MethodGet, c.FileURL(p, "stat"), nil, &l); err != nil {
		return nil, err
	}
	if l.Children == nil {
		l.Children = []Node{}
	}
	return &l, nil
}

func (c *Client) action(ctx context.Context, p string, form url.Values, out any) error {
	return c.doJSON(ctx, http.MethodPost, c.FileURL(p, ""), form, out)
}

func (c *Client) Mkdir(ctx context.Context, p string) error {
	return c.action(ctx, p, url.Values{"action": {"mkdir"}}, nil)
}

func (c *Client) MkdirAll(ctx context.Context, p string) error {
	return c.action(ctx, p, url.Values{"action": {"mkdirall"}}, nil)
}

func (c *Client) Rename(ctx context.Context, from, to string) error {
	return c.action(ctx, from, url.Values{"action": {"rename"}, "target": {to}}, nil)
}

// CopyFile copies a single file. The API has no server-side copy, so the
// contents are streamed down and straight back up.
func (c *Client) CopyFile(ctx context.Context, from, to string, progress func(io.Reader) io.Reader) error {
	res, err := c.Do(ctx, http.MethodGet, c.FileURL(from, ""), nil, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var body io.Reader = res.Body
	if progress != nil {
		body = progress(body)
	}
	ct := res.Header.Get("Content-Type")
	return c.Upload(ctx, to, body, res.ContentLength, ct)
}

// SetShared toggles the public link of a node and returns the updated node.
func (c *Client) SetShared(ctx context.Context, p string, shared bool) (*Node, error) {
	var n Node
	err := c.action(ctx, p, url.Values{"action": {"update"}, "shared": {strconv.FormatBool(shared)}}, &n)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (c *Client) Delete(ctx context.Context, p string, recursive bool) error {
	q := ""
	if recursive {
		q = "recursive"
	}
	return c.doJSON(ctx, http.MethodDelete, c.FileURL(p, q), nil, nil)
}

// Search returns paths below p matching term.
func (c *Client) Search(ctx context.Context, p, term string, limit int) ([]string, error) {
	q := url.Values{"search": {term}, "limit": {strconv.Itoa(limit)}}
	var out []string
	if err := c.doJSON(ctx, http.MethodGet, c.FileURL(p, q.Encode()), nil, &out); err != nil {
		return nil, err
	}
	for i, s := range out {
		out[i] = CleanPath(s)
	}
	return out, nil
}

// Upload streams body to the file at p.
func (c *Client) Upload(ctx context.Context, p string, body io.Reader, size int64, contentType string) error {
	req, err := c.newRequest(ctx, http.MethodPut, c.FileURL(p, ""), body)
	if err != nil {
		return err
	}
	req.ContentLength = size
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := c.transfer.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return decodeError(res)
	}
	return nil
}

// CleanPath collapses duplicate slashes and strips trailing ones.
func CleanPath(p string) string {
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		switch s {
		case "", ".":
			continue
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			continue
		}
		out = append(out, s)
	}
	return "/" + strings.Join(out, "/")
}

// Join joins a directory and a name into a clean path.
func Join(dir, name string) string {
	return CleanPath(dir + "/" + name)
}

// Parent returns the parent directory of p.
func Parent(p string) string {
	p = CleanPath(p)
	i := strings.LastIndex(p, "/")
	if i <= 0 {
		return "/"
	}
	return p[:i]
}

// Base returns the last element of p.
func Base(p string) string {
	p = CleanPath(p)
	return p[strings.LastIndex(p, "/")+1:]
}
