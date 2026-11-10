package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const sessionCookie = "crosswatc_session"
const transactionCookie = "crosswatc_login"
const logoutEvent = "http://schemas.openid.net/event/backchannel-logout"

// BrowserConfig enables the confidential OIDC browser client alongside bearer API access.
type BrowserConfig struct {
	ClientID, ClientSecret, PublicURL string
	SessionTTL                        time.Duration
}
type loginTransaction struct {
	binding, nonce, verifier string
	expires                  time.Time
}
type browserSession struct {
	mu                    sync.Mutex
	token                 *oauth2.Token
	idToken, subject, sid string
	expires               time.Time
	ctx                   context.Context
	cancel                context.CancelFunc
}

// Browser keeps tokens on the server; cookies contain only random, revocable identifiers.
// ponytail: process-local sessions require one API replica; use a shared session store plus
// revocation fan-out before scaling replicas. Restarting the API signs browser sessions out.
type Browser struct {
	api                    *OIDC
	oauth                  oauth2.Config
	verifier               *oidc.IDTokenVerifier
	origin, logoutEndpoint string
	secure                 bool
	ttl                    time.Duration
	client                 *http.Client
	mu                     sync.Mutex
	transactions           map[[32]byte]loginTransaction
	sessions               map[[32]byte]*browserSession
	logoutIDs              map[string]time.Time
}

func NewBrowser(api *OIDC, cfg BrowserConfig) (*Browser, error) {
	u, err := url.Parse(cfg.PublicURL)
	if api.provider == nil || err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.SessionTTL <= 0 || cfg.SessionTTL > 24*time.Hour {
		return nil, errors.New("invalid browser OIDC configuration")
	}
	var metadata struct {
		Logout string `json:"end_session_endpoint"`
	}
	if err := api.provider.Claims(&metadata); err != nil {
		return nil, err
	}
	if metadata.Logout != "" {
		endpoint, err := url.Parse(metadata.Logout)
		if err != nil || endpoint.Host == "" || (endpoint.Scheme != "https" && u.Scheme == "https") {
			return nil, errors.New("invalid OIDC logout endpoint")
		}
	}
	return &Browser{api: api, oauth: oauth2.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, Endpoint: api.provider.Endpoint(), RedirectURL: cfg.PublicURL + "/api/v1/auth/callback", Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}, verifier: api.provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}), origin: cfg.PublicURL, logoutEndpoint: metadata.Logout, secure: u.Scheme == "https", ttl: cfg.SessionTTL, client: &http.Client{Timeout: 10 * time.Second}, transactions: make(map[[32]byte]loginTransaction), sessions: make(map[[32]byte]*browserSession), logoutIDs: make(map[string]time.Time)}, nil
}

func randomID() string                 { return rand.Text() + rand.Text() }
func cookieHash(value string) [32]byte { return sha256.Sum256([]byte(value)) }
func (b *Browser) cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/api/v1", MaxAge: age, HttpOnly: true, Secure: b.secure, SameSite: http.SameSiteLaxMode})
}
func authError(w http.ResponseWriter, status int) {
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, "Authentication request could not be completed", status)
}
func (b *Browser) cleanLocked() {
	now := time.Now()
	for key, tx := range b.transactions {
		if !now.Before(tx.expires) {
			delete(b.transactions, key)
		}
	}
	for key, s := range b.sessions {
		if !now.Before(s.expires) {
			s.cancel()
			delete(b.sessions, key)
		}
	}
	for key, expiry := range b.logoutIDs {
		if !now.Before(expiry) {
			delete(b.logoutIDs, key)
		}
	}
}
func (b *Browser) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	state, binding, nonce := randomID(), randomID(), randomID()
	tx := loginTransaction{binding: binding, nonce: nonce, verifier: oauth2.GenerateVerifier(), expires: time.Now().Add(5 * time.Minute)}
	b.mu.Lock()
	b.cleanLocked()
	if len(b.transactions) >= 1000 {
		b.mu.Unlock()
		authError(w, 429)
		return
	}
	b.transactions[cookieHash(state)] = tx
	b.mu.Unlock()
	b.cookie(w, transactionCookie, binding, 300)
	http.Redirect(w, r, b.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(tx.verifier)), http.StatusFound)
}
func (b *Browser) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	binding, err := r.Cookie(transactionCookie)
	state := r.URL.Query().Get("state")
	b.mu.Lock()
	tx, ok := b.transactions[cookieHash(state)]
	delete(b.transactions, cookieHash(state))
	b.mu.Unlock()
	b.cookie(w, transactionCookie, "", -1)
	if err != nil || state == "" || !ok || !time.Now().Before(tx.expires) || subtle.ConstantTimeCompare([]byte(binding.Value), []byte(tx.binding)) != 1 || r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		authError(w, 400)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, b.client)
	token, err := b.oauth.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(tx.verifier))
	if err != nil {
		authError(w, 401)
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		authError(w, 401)
		return
	}
	id, err := b.verifier.Verify(ctx, raw)
	if err != nil || id.Subject == "" || subtle.ConstantTimeCompare([]byte(id.Nonce), []byte(tx.nonce)) != 1 {
		authError(w, 401)
		return
	}
	access, err := b.api.verifier.Verify(ctx, token.AccessToken)
	if err != nil || access.Subject != id.Subject || access.Issuer != id.Issuer {
		authError(w, 401)
		return
	}
	var claims struct {
		SID string `json:"sid"`
	}
	if id.Claims(&claims) != nil {
		authError(w, 401)
		return
	}
	if _, err = b.api.principal(access); err != nil {
		authError(w, 401)
		return
	}
	token.Expiry = access.Expiry
	// Rotate the browser session when signing in again.
	b.remove(r)
	key := randomID()
	expiry := time.Now().Add(b.ttl)
	sessionCtx, sessionCancel := context.WithDeadline(context.Background(), expiry)
	s := &browserSession{token: token, idToken: raw, subject: id.Subject, sid: claims.SID, expires: expiry, ctx: sessionCtx, cancel: sessionCancel}
	b.mu.Lock()
	b.cleanLocked()
	if len(b.sessions) >= 10000 {
		b.mu.Unlock()
		sessionCancel()
		authError(w, 429)
		return
	}
	b.sessions[cookieHash(key)] = s
	b.mu.Unlock()
	b.cookie(w, sessionCookie, key, int(b.ttl.Seconds()))
	http.Redirect(w, r, b.origin+"/dashboard", http.StatusFound)
}
func (b *Browser) session(r *http.Request) *browserSession {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[cookieHash(c.Value)]
	if s != nil && !time.Now().Before(s.expires) {
		s.cancel()
		delete(b.sessions, cookieHash(c.Value))
		return nil
	}
	return s
}
func (b *Browser) remove(r *http.Request) *browserSession {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	key := cookieHash(c.Value)
	s := b.sessions[key]
	delete(b.sessions, key)
	if s != nil {
		s.cancel()
	}
	return s
}

// Bind closes long-lived streams when logout/backchannel revokes their session.
func (b *Browser) Bind(r *http.Request) func() {
	if r.Header.Get("Authorization") != "" {
		return func() {}
	}
	s := b.session(r)
	if s == nil {
		return func() {}
	}
	s.mu.Lock()
	expiry := s.token.Expiry
	s.mu.Unlock()
	ctx, cancel := context.WithDeadline(r.Context(), expiry)
	stop := context.AfterFunc(s.ctx, cancel)
	*r = *r.WithContext(ctx)
	return func() { stop(); cancel() }
}
func (b *Browser) Authenticate(r *http.Request) (Principal, error) {
	if r.Header.Get("Authorization") != "" {
		return b.api.Authenticate(r)
	}
	s := b.session(r)
	if s == nil {
		return Principal{}, ErrUnauthenticated
	}
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && r.Header.Get("Origin") != b.origin {
		return Principal{}, ErrUnauthenticated
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx.Err() != nil {
		return Principal{}, ErrUnauthenticated
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, b.client)
	// The OAuth2 token source refreshes server-side. No token ever enters browser storage.
	token, err := b.oauth.TokenSource(ctx, s.token).Token()
	if err != nil {
		b.remove(r)
		return Principal{}, ErrUnauthenticated
	}
	access, err := b.api.verifier.Verify(ctx, token.AccessToken)
	if err != nil || access.Subject != s.subject {
		b.remove(r)
		return Principal{}, ErrUnauthenticated
	}
	if raw, ok := token.Extra("id_token").(string); token.AccessToken != s.token.AccessToken && ok && raw != "" {
		id, err := b.verifier.Verify(ctx, raw)
		if err != nil || id.Subject != s.subject {
			b.remove(r)
			return Principal{}, ErrUnauthenticated
		}
		s.idToken = raw
	}
	token.Expiry = access.Expiry
	s.token = token
	return b.api.principal(access)
}
func (b *Browser) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != b.origin {
		authError(w, 403)
		return
	}
	s := b.remove(r)
	b.cookie(w, sessionCookie, "", -1)
	b.cookie(w, transactionCookie, "", -1)
	target := b.origin
	if b.logoutEndpoint != "" && s != nil {
		s.mu.Lock()
		idToken := s.idToken
		s.mu.Unlock()
		u, _ := url.Parse(b.logoutEndpoint)
		q := u.Query()
		q.Set("client_id", b.oauth.ClientID)
		q.Set("id_token_hint", idToken)
		q.Set("post_logout_redirect_uri", b.origin)
		u.RawQuery = q.Encode()
		target = u.String()
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"logoutUrl": target})
}
func (b *Browser) BackchannelLogout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if r.ParseForm() != nil {
		authError(w, 400)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	token, err := b.verifier.VerifyLogout(ctx, r.PostForm.Get("logout_token"))
	if err != nil || token.IssuedAt.IsZero() || time.Since(token.IssuedAt) > 5*time.Minute || token.IssuedAt.After(time.Now().Add(30*time.Second)) {
		authError(w, 400)
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanLocked()
	if _, replayed := b.logoutIDs[token.TokenID]; replayed {
		authError(w, 400)
		return
	}
	if len(b.logoutIDs) >= 10000 {
		authError(w, 429)
		return
	}
	b.logoutIDs[token.TokenID] = time.Now().Add(5 * time.Minute)
	for key, s := range b.sessions {
		if (token.SessionID != "" && s.sid == token.SessionID && (token.Subject == "" || s.subject == token.Subject)) || (token.SessionID == "" && s.subject == token.Subject) {
			s.cancel()
			delete(b.sessions, key)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}
