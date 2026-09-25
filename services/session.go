// Package services holds the Wails-facing adapters. Business logic lives in
// internal/ packages; these types only translate between the UI and them.
package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"nova/internal/config"
	"nova/internal/nova"
)

// Session is the signed-in state shown to the UI.
type Session struct {
	SignedIn bool       `json:"signedIn"`
	Server   string     `json:"server"`
	User     *nova.User `json:"user"`
}

// SessionService manages credentials and preferences.
type SessionService struct {
	client *nova.Client
	store  *config.Store
}

func NewSessionService(client *nova.Client, store *config.Store) *SessionService {
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
	out.SignedIn, out.User = true, u
	return out, nil
}

// SignInWithKey signs in using an existing API key.
func (s *SessionService) SignInWithKey(key string) (Session, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Session{}, errors.New("enter an API key")
	}
	return s.activate(key, false)
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
	return s.activate(key, true)
}

func (s *SessionService) activate(key string, fromLogin bool) (Session, error) {
	prevKey := s.store.Get().APIKey
	s.client.SetAPIKey(key)
	ctx, cancel := ctxTimeout()
	defer cancel()
	u, err := s.client.User(ctx)
	if err != nil {
		s.client.SetAPIKey(prevKey)
		return Session{}, err
	}
	if err := s.store.Update(func(c *config.Config) {
		c.APIKey = key
		c.KeyFromLogin = fromLogin
	}); err != nil {
		return Session{}, err
	}
	return Session{SignedIn: true, Server: s.client.BaseURL(), User: u}, nil
}

// Refresh reloads the user info (quota etc).
func (s *SessionService) Refresh() (*nova.User, error) {
	ctx, cancel := ctxTimeout()
	defer cancel()
	return s.client.User(ctx)
}

// SignOut forgets the stored key. Keys created by SignIn are revoked;
// keys the user pasted in are left alone since they may be used elsewhere.
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
	_ = s.store.Update(func(c *config.Config) { c.APIKey, c.KeyFromLogin = "", false })
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
