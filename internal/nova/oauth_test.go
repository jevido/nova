package nova

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCodeChallenge(t *testing.T) {
	// The test values of RFC 7636 appendix B, also on nova.storage's page.
	got := codeChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	if want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"; got != want {
		t.Fatalf("challenge = %q, want %q", got, want)
	}
	v, err := randomToken()
	if err != nil || len(v) != 43 || strings.ContainsAny(v, "+/=") {
		t.Fatalf("verifier %q (%v) is not 43 URL-safe characters", v, err)
	}
}

// fakeOAuth is a server with nova.storage's token endpoint. Its "browser"
// approves (or denies) the request at once by calling the redirect URI.
type fakeOAuth struct {
	t         *testing.T
	srv       *httptest.Server
	deny      bool
	challenge string
	redirect  string
	dirs      []string
}

func newFakeOAuth(t *testing.T) *fakeOAuth {
	f := &fakeOAuth{t: t, dirs: []string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/oauth/token" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("token request has an Authorization header")
		}
		r.ParseForm()
		ok := r.Form.Get("grant_type") == "authorization_code" &&
			r.Form.Get("code") == "the-code" &&
			r.Form.Get("redirect_uri") == f.redirect &&
			codeChallenge(r.Form.Get("code_verifier")) == f.challenge
		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant", "error_description": "The code is not valid"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "the-key", "token_type": "Bearer",
			"scope": r.Form.Get("scope"), "filesystem_dirs": f.dirs,
		})
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeOAuth) browser(u string) error {
	au, err := url.Parse(u)
	if err != nil {
		return err
	}
	q := au.Query()
	if au.Path != "/oauth/authorize" || q.Get("response_type") != "code" || q.Get("client_id") != OAuthClientID ||
		q.Get("scope") != OAuthScope || q.Get("code_challenge_method") != "S256" || len(q.Get("code_challenge")) != 43 {
		f.t.Errorf("unexpected authorization URL %s", u)
	}
	if q.Has("filesystem_dirs") {
		f.t.Error("authorization URL has filesystem_dirs")
	}
	f.challenge, f.redirect = q.Get("code_challenge"), q.Get("redirect_uri")
	if !strings.HasPrefix(f.redirect, "http://127.0.0.1:") || !strings.HasSuffix(f.redirect, "/oauth/callback") {
		f.t.Errorf("redirect URI %q is not a 127.0.0.1 loopback", f.redirect)
	}
	go func() {
		// A request with a wrong state is ignored.
		if res, err := http.Get(f.redirect + "?code=evil&state=wrong"); err == nil {
			res.Body.Close()
		}
		back := url.Values{"state": {q.Get("state")}}
		if f.deny {
			back.Set("error", "access_denied")
		} else {
			back.Set("code", "the-code")
		}
		if res, err := http.Get(f.redirect + "?" + back.Encode()); err == nil {
			res.Body.Close()
		}
	}()
	return nil
}

func TestOAuthLogin(t *testing.T) {
	f := newFakeOAuth(t)
	f.dirs = []string{"6m6r7854Ubzt491q4AJuCb"}
	c := NewClient(f.srv.URL, "old-key")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tok, err := c.OAuthLogin(ctx, OAuthClientID, OAuthScope, f.browser)
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "the-key" || len(tok.FilesystemDirs) != 1 || tok.FilesystemDirs[0] != f.dirs[0] {
		t.Fatalf("unexpected token %+v", tok)
	}
}

func TestOAuthLoginDenied(t *testing.T) {
	f := newFakeOAuth(t)
	f.deny = true
	c := NewClient(f.srv.URL, "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.OAuthLogin(ctx, OAuthClientID, OAuthScope, f.browser); !errors.Is(err, ErrSignInDenied) {
		t.Fatalf("err = %v, want ErrSignInDenied", err)
	}
}

func TestOAuthLoginCancel(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "")
	ctx, cancel := context.WithCancel(context.Background())
	// The user never comes back from the browser.
	_, err := c.OAuthLogin(ctx, OAuthClientID, OAuthScope, func(string) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestOAuthTokenRetriesUntilReachable(t *testing.T) {
	f := newFakeOAuth(t)
	// The server can't be reached at first, like a phone app in the
	// background, then comes back.
	real := f.srv.Listener.Addr().String()
	c := NewClient("http://nova-storage.invalid", "")
	f.challenge, f.redirect = codeChallenge("v"), "http://127.0.0.1:1/oauth/callback"
	go func() {
		time.Sleep(1200 * time.Millisecond)
		c.SetBaseURL("http://" + real)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tok, err := c.oauthTokenRetrying(ctx, "the-code", f.redirect, "v")
	if err != nil || tok.AccessToken != "the-key" {
		t.Fatalf("tok %+v, err %v", tok, err)
	}
	// A refused code is not retried.
	if _, err := c.oauthTokenRetrying(ctx, "wrong", f.redirect, "v"); err == nil || notSent(err) {
		t.Fatalf("err = %v, want the server's refusal", err)
	}
}

func TestCallbackPageClosesTab(t *testing.T) {
	if !strings.Contains(callbackPage, "window.close()") {
		t.Fatal("the callback page doesn't try to close its tab")
	}
}
