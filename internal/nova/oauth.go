package nova

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// OAuth sign-in: the authorization code flow with PKCE and a loopback
// redirect (RFC 6749, 7636 and 8252), as nova.storage offers it to apps. The
// user approves Nova in their own browser, so Nova never sees their password,
// and gets back an ordinary API key with only the permissions asked for.

// OAuthClientID is the ID Nova is registered with on nova.storage.
const OAuthClientID = "jevido-nova"

// OAuthScope is what a file manager needs: the files, and the account
// information for the sidebar (user name, plan, storage used).
const OAuthScope = "filesystem_read filesystem_write account_read"

// oauthCallbackPath is the path of the registered redirect URI.
const oauthCallbackPath = "/oauth/callback"

// OAuthToken is the answer of the token endpoint.
type OAuthToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	// FilesystemDirs are the IDs of the directories the user limited the key
	// to. Empty means the whole account. The key can't tell this later, so
	// it has to be stored with the key.
	FilesystemDirs []string `json:"filesystem_dirs"`
}

// ErrSignInDenied is returned when the user chose Deny in the browser.
var ErrSignInDenied = errors.New("sign-in was denied in the browser")

// randomToken returns 32 random bytes as 43 URL-safe characters, good for a
// PKCE code verifier and for the state.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// codeChallenge is the S256 PKCE challenge of verifier.
func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// callbackPage is what the browser shows once it has handed Nova the answer.
const callbackPage = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Nova</title>
<style>body{font-family:system-ui,sans-serif;display:grid;place-items:center;height:100vh;margin:0;background:#fafafb;color:#222}
@media(prefers-color-scheme:dark){body{background:#222226;color:#fff}}
div{text-align:center}h1{font-size:1.6em;margin:0 0 .4em}p{opacity:.7;margin:0}</style></head>
<body><div><h1>%s</h1><p>You can close this tab and go back to Nova.</p></div></body></html>`

// OAuthLogin signs the user in through their browser and returns the new
// key. openBrowser must open the URL in the user's default browser, not in a
// web view. The user may never come back (they can close the tab, and an
// invalid request ends on an error page of Nova), so cancel ctx to give up.
func (c *Client) OAuthLogin(ctx context.Context, clientID, scope string, openBrowser func(string) error) (*OAuthToken, error) {
	verifier, err := randomToken()
	if err != nil {
		return nil, err
	}
	state, err := randomToken()
	if err != nil {
		return nil, err
	}

	// Any free port, on 127.0.0.1: the registration doesn't accept localhost.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("could not wait for the browser: %w", err)
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d%s", listener.Addr().(*net.TCPAddr).Port, oauthCallbackPath)

	type answer struct {
		code string
		err  error
	}
	answers := make(chan answer, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != oauthCallbackPath || q.Get("state") != state {
			// Not the answer to this sign-in; keep waiting.
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		var a answer
		title := "Signed in to Nova"
		switch {
		case q.Get("error") == "access_denied":
			a.err, title = ErrSignInDenied, "Sign-in denied"
		case q.Get("error") != "":
			a.err, title = fmt.Errorf("sign-in failed: %s", q.Get("error")), "Sign-in failed"
		case q.Get("code") == "":
			a.err, title = errors.New("sign-in failed: the browser sent no code"), "Sign-in failed"
		default:
			a.code = q.Get("code")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, callbackPage, title)
		select {
		case answers <- a:
		default: // a second callback; the first one counts
		}
	})}
	go server.Serve(listener)
	defer server.Close()

	authURL := c.BaseURL() + "/oauth/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {scope},
		"state":                 {state},
		"code_challenge":        {codeChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}.Encode()
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("could not open the browser: %w", err)
	}

	var a answer
	select {
	case a = <-answers:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if a.err != nil {
		return nil, a.err
	}
	return c.oauthToken(ctx, a.code, redirectURI, verifier)
}

// oauthToken trades an authorization code for the key.
func (c *Client) oauthToken(ctx context.Context, code, redirectURI, verifier string) (*OAuthToken, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	// Not newRequest: this request must not carry an API key, which the
	// server would take for one.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL()+"/api/user/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "nova-desktop/0.1")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body := io.LimitReader(res.Body, 64<<10)
	if res.StatusCode != http.StatusOK {
		var e struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		_ = json.NewDecoder(body).Decode(&e)
		if e.Description != "" {
			return nil, fmt.Errorf("sign-in failed: %s", e.Description)
		}
		if e.Error != "" {
			return nil, fmt.Errorf("sign-in failed: %s", strings.ReplaceAll(e.Error, "_", " "))
		}
		return nil, fmt.Errorf("sign-in failed with status %d", res.StatusCode)
	}
	var t OAuthToken
	if err := json.NewDecoder(body).Decode(&t); err != nil {
		return nil, err
	}
	if t.AccessToken == "" {
		return nil, errors.New("sign-in failed: the server sent no key")
	}
	if t.FilesystemDirs == nil {
		t.FilesystemDirs = []string{}
	}
	return &t, nil
}
