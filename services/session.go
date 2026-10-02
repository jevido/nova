// Package services holds the Wails-facing adapters. Business logic lives in
// internal/ packages; these types only translate between the UI and them.
package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"nova/internal/config"
	"nova/internal/nova"
)

// Session is the signed-in state shown to the UI.
type Session struct {
	SignedIn bool       `json:"signedIn"`
	Server   string     `json:"server"`
	User     *nova.User `json:"user"`
	// Roots are the folders the key is limited to, shown in place of Home.
	// Empty when Nova can reach the whole account.
	Roots []Root `json:"roots"`
}

// Root is one of the folders a limited key can reach.
type Root struct {
	Path string `json:"path"` // "/{id}"
	Name string `json:"name"`
}

// SessionService manages credentials and preferences.
type SessionService struct {
	client *nova.Client
	store  *config.Store

	// cancelBrowser stops a browser sign-in that is waiting for the user.
	browserMu     sync.Mutex
	cancelBrowser context.CancelFunc
}

// browserSignInTimeout is how long a browser sign-in waits for the user.
const browserSignInTimeout = 10 * time.Minute

func NewSessionService(client *nova.Client, store *config.Store) *SessionService {
	setRoots(store.Get().Roots)
	return &SessionService{client: client, store: store}
}

func ctxTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// Restore validates the saved key, if any, and returns the session.
func (s *SessionService) Restore() (Session, error) {
	out := Session{Server: s.client.BaseURL()}
	if s.client.APIKey() == "" {
		return out, nil
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	u, err := s.client.User(ctx)
	if errors.Is(err, nova.ErrUnauthorized) {
		s.forget()
		return out, nil
	}
	if err != nil {
		// Offline or server down: keep the key, report the error.
		return out, err
	}
	out.SignedIn, out.User, out.Roots = true, u, s.rootList(ctx)
	return out, nil
}

// rootList names the folders the key is limited to.
func (s *SessionService) rootList(ctx context.Context) []Root {
	out := []Root{}
	for _, p := range topFolders() {
		if p == HomeDir {
			continue
		}
		r := Root{Path: p, Name: "Folder"}
		if l, err := s.client.Stat(ctx, p); err == nil && len(l.Path) > 0 && l.Path[0].Name != "" {
			r.Name = l.Path[0].Name
		}
		out = append(out, r)
	}
	return out
}

// SignInWithKey signs in using an existing API key.
func (s *SessionService) SignInWithKey(key string) (Session, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Session{}, errors.New("enter an API key")
	}
	return s.activate(key, false, nil)
}

// SignIn exchanges a username and password for an API key.
func (s *SessionService) SignIn(username, password string) (Session, error) {
	if strings.TrimSpace(username) == "" || password == "" {
		return Session{}, errors.New("enter your username and password")
	}
	prevKey := s.client.APIKey()
	s.client.SetAPIKey("")
	ctx, cancel := ctxTimeout()
	defer cancel()
	key, err := s.client.Login(ctx, strings.TrimSpace(username), password, "Nova Desktop")
	if err != nil {
		s.client.SetAPIKey(prevKey)
		return Session{}, err
	}
	return s.activate(key, true, nil)
}

// SignInWithBrowser signs in on the nova.storage website, in the user's own
// browser, where they approve Nova and may limit it to some folders. It
// waits until they have answered, CancelBrowserSignIn is called, or ten
// minutes have passed.
func (s *SessionService) SignInWithBrowser() (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), browserSignInTimeout)
	defer cancel()
	s.browserMu.Lock()
	if s.cancelBrowser != nil {
		s.cancelBrowser() // only the newest sign-in waits
	}
	s.cancelBrowser = cancel
	s.browserMu.Unlock()
	defer func() {
		s.browserMu.Lock()
		s.cancelBrowser = nil
		s.browserMu.Unlock()
	}()

	tok, err := s.client.OAuthLogin(ctx, nova.OAuthClientID, nova.OAuthScope, openURL)
	switch {
	case errors.Is(err, context.Canceled):
		return Session{}, errors.New("sign-in was cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return Session{}, errors.New("sign-in timed out; try again")
	case err != nil:
		return Session{}, err
	}
	return s.activate(tok.AccessToken, true, tok.FilesystemDirs)
}

// CancelBrowserSignIn stops waiting for the browser.
func (s *SessionService) CancelBrowserSignIn() {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()
	if s.cancelBrowser != nil {
		s.cancelBrowser()
	}
}

// activate checks and stores key. dirs are the folder IDs it is limited to.
func (s *SessionService) activate(key string, fromLogin bool, dirs []string) (Session, error) {
	prev := s.store.Get()
	s.client.SetAPIKey(key)
	setRoots(dirs)
	ctx, cancel := ctxTimeout()
	defer cancel()
	u, err := s.client.User(ctx)
	if err != nil {
		s.client.SetAPIKey(prev.APIKey)
		setRoots(prev.Roots)
		return Session{}, err
	}
	if err := s.store.Update(func(c *config.Config) {
		c.APIKey = key
		c.KeyFromLogin = fromLogin
		c.Roots = dirs
	}); err != nil {
		return Session{}, err
	}
	return Session{SignedIn: true, Server: s.client.BaseURL(), User: u, Roots: s.rootList(ctx)}, nil
}

// Refresh reloads the user info (quota etc).
func (s *SessionService) Refresh() (*nova.User, error) {
	ctx, cancel := ctxTimeout()
	defer cancel()
	return s.client.User(ctx)
}

// SignOut forgets the stored key. Keys created by SignIn or
// SignInWithBrowser are revoked; keys the user pasted in are left alone
// since they may be used elsewhere.
func (s *SessionService) SignOut() error {
	if s.store.Get().KeyFromLogin {
		ctx, cancel := ctxTimeout()
		_ = s.client.Logout(ctx)
		cancel()
	}
	s.forget()
	return nil
}

func (s *SessionService) forget() {
	s.client.SetAPIKey("")
	setRoots(nil)
	_ = s.store.Update(func(c *config.Config) { c.APIKey, c.KeyFromLogin, c.Roots = "", false, nil })
}

func (s *SessionService) Prefs() config.Prefs { return s.store.Get().Prefs }

func (s *SessionService) SavePrefs(p config.Prefs) error {
	if p.Bookmarks == nil {
		p.Bookmarks = []config.Bookmark{}
	}
	if p.Starred == nil {
		p.Starred = []string{}
	}
	return s.store.Update(func(c *config.Config) { c.Prefs = p })
}
