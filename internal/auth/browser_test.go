package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type fakeBrowserProvider struct {
	key                              *rsa.PrivateKey
	issuer, nonce, challenge, access string
	invalid                          string
	refreshes                        int
}

func browserFixture(t *testing.T) (*Browser, *fakeBrowserProvider) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeBrowserProvider{key: key}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": f.issuer, "authorization_endpoint": f.issuer + "/authorize", "token_endpoint": f.issuer + "/token", "jwks_uri": f.issuer + "/keys", "end_session_endpoint": "https://keycloak.example/logout", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			enc := base64.RawURLEncoding.EncodeToString
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "kid": "k1", "use": "sig", "alg": "RS256", "n": enc(f.key.N.Bytes()), "e": enc(big.NewInt(int64(f.key.E)).Bytes())}}})
		case "/token":
			_ = r.ParseForm()
			if r.PostForm.Get("grant_type") == "refresh_token" {
				f.refreshes++
			} else if r.PostForm.Get("code_verifier") == "" || s256(r.PostForm.Get("code_verifier")) != f.challenge {
				http.Error(w, "PKCE required", 400)
				return
			}
			claims := map[string]any{"iss": f.issuer, "sub": "stable-subject", "aud": "browser-client", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "nonce": f.nonce, "sid": "keycloak-session"}
			switch f.invalid {
			case "nonce":
				claims["nonce"] = "wrong"
			case "issuer":
				claims["iss"] = "https://other.example"
			case "audience":
				claims["aud"] = "other"
			case "expired":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "subject":
				claims["sub"] = "different"
			}
			accessClaims := map[string]any{"iss": f.issuer, "sub": "stable-subject", "aud": "api-audience", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "preferred_username": "admin", "groups": []string{"squad-payments"}}
			f.access = signedJWT(t, f.key, accessClaims)
			response := map[string]any{"access_token": f.access, "refresh_token": "server-side-refresh", "token_type": "Bearer", "expires_in": 3600}
			if f.invalid != "missing-id" {
				response["id_token"] = signedJWT(t, f.key, claims)
			}
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	f.issuer = server.URL
	t.Cleanup(server.Close)
	provider, err := oidc.NewProvider(context.Background(), f.issuer)
	if err != nil {
		t.Fatal(err)
	}
	api := &OIDC{provider: provider, verifier: provider.Verifier(&oidc.Config{ClientID: "api-audience"}), mapping: GroupMapping{SquadPrefix: "squad-", SREGroup: "platform-sre", AdminGroup: "platform-admin", RolesClientID: "api-audience"}}
	b, err := NewBrowser(api, BrowserConfig{ClientID: "browser-client", ClientSecret: "fixture-secret", PublicURL: "https://workspace.example", SessionTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for _, s := range b.sessions {
			s.cancel()
		}
	})
	return b, f
}
func s256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func beginLogin(t *testing.T, b *Browser, f *fakeBrowserProvider) (string, *http.Cookie) {
	t.Helper()
	w := httptest.NewRecorder()
	b.Login(w, httptest.NewRequest("GET", "https://workspace.example/api/v1/auth/login", nil))
	if w.Code != 302 {
		t.Fatalf("login: %d", w.Code)
	}
	u, _ := url.Parse(w.Header().Get("Location"))
	q := u.Query()
	f.nonce = q.Get("nonce")
	f.challenge = q.Get("code_challenge")
	if f.nonce == "" || f.challenge == "" || q.Get("code_challenge_method") != "S256" || q.Get("redirect_uri") != "https://workspace.example/api/v1/auth/callback" {
		t.Fatal("invalid login parameters")
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatal("unsafe login cookie")
	}
	return q.Get("state"), cookies[0]
}
func callback(t *testing.T, b *Browser, state string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://workspace.example/api/v1/auth/callback?code=fixture&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	b.Callback(w, r)
	return w
}
func signedIn(t *testing.T, b *Browser, f *fakeBrowserProvider) *http.Cookie {
	t.Helper()
	state, cookie := beginLogin(t, b, f)
	w := callback(t, b, state, cookie)
	if w.Code != 302 {
		t.Fatalf("callback: %d %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			if strings.Contains(c.Value, ".") || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
				t.Fatal("unsafe session cookie")
			}
			return c
		}
	}
	t.Fatal("missing session cookie")
	return nil
}
func TestBrowserLoginValidatesTransactionAndTokens(t *testing.T) {
	b, f := browserFixture(t)
	for _, mode := range []string{"nonce", "issuer", "audience", "expired", "subject", "missing-id"} {
		t.Run(mode, func(t *testing.T) {
			f.invalid = mode
			state, cookie := beginLogin(t, b, f)
			if w := callback(t, b, state, cookie); w.Code != 401 {
				t.Fatalf("accepted invalid %s: %d", mode, w.Code)
			}
		})
	}
	f.invalid = ""
	state, cookie := beginLogin(t, b, f)
	if w := callback(t, b, state, nil); w.Code != 400 {
		t.Fatal("accepted missing binding")
	}
	state, cookie = beginLogin(t, b, f)
	if w := callback(t, b, "wrong-state", cookie); w.Code != 400 {
		t.Fatal("accepted wrong state")
	}
	b.mu.Lock()
	tx := b.transactions[cookieHash(state)]
	tx.expires = time.Now().Add(-time.Second)
	b.transactions[cookieHash(state)] = tx
	b.mu.Unlock()
	if w := callback(t, b, state, cookie); w.Code != 400 {
		t.Fatal("accepted expired state")
	}
	state, cookie = beginLogin(t, b, f)
	if w := callback(t, b, state, cookie); w.Code != 302 {
		t.Fatal("valid callback rejected")
	}
	if w := callback(t, b, state, cookie); w.Code != 400 {
		t.Fatal("accepted replay")
	}
}
func TestBrowserSessionCSRFRefreshAndLogout(t *testing.T) {
	b, f := browserFixture(t)
	cookie := signedIn(t, b, f)
	request := func(method string) *http.Request {
		r := httptest.NewRequest(method, "https://workspace.example/api/v1/me", nil)
		r.AddCookie(cookie)
		return r
	}
	r := request("GET")
	p, err := b.Authenticate(r)
	if err != nil || p.Admin || p.User != "admin" || p.Subject != "stable-subject" || p.Identity() != f.issuer+"#stable-subject" || !p.CanWrite("payments") {
		t.Fatalf("bad identity: %+v %v", p, err)
	}
	if _, err := b.Authenticate(request("POST")); err == nil {
		t.Fatal("mutation without origin allowed")
	}
	r = request("DELETE")
	r.Header.Set("Origin", "https://evil.example")
	if _, err := b.Authenticate(r); err == nil {
		t.Fatal("cross-origin mutation allowed")
	}
	r = request("POST")
	r.Header.Set("Origin", b.origin)
	if _, err := b.Authenticate(r); err != nil {
		t.Fatal(err)
	}
	r = request("GET")
	r.Header.Set("Authorization", "Bearer invalid")
	if _, err := b.Authenticate(r); err == nil {
		t.Fatal("invalid bearer fell back to cookie")
	}
	s := b.session(request("GET"))
	s.mu.Lock()
	s.token.Expiry = time.Now().Add(-time.Second)
	s.mu.Unlock()
	if _, err := b.Authenticate(request("GET")); err != nil || f.refreshes != 1 {
		t.Fatalf("refresh: %v count=%d", err, f.refreshes)
	}
	r = request("GET")
	release := b.Bind(r)
	defer release()
	w := httptest.NewRecorder()
	b.Logout(w, request("POST"))
	if w.Code != 403 {
		t.Fatal("logout without origin allowed")
	}
	logout := request("POST")
	logout.Header.Set("Origin", b.origin)
	w = httptest.NewRecorder()
	b.Logout(w, logout)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "logoutUrl") {
		t.Fatal("logout failed")
	}
	select {
	case <-r.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("active stream not revoked")
	}
	if _, err := b.Authenticate(request("GET")); err == nil {
		t.Fatal("logged out cookie accepted")
	}
}
func TestBrowserBackchannelRejectsInvalidTokensAndRevokes(t *testing.T) {
	b, f := browserFixture(t)
	cookie := signedIn(t, b, f)
	claims := map[string]any{"iss": f.issuer, "aud": "browser-client", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(), "jti": "logout-1", "sid": "keycloak-session", "events": map[string]any{logoutEvent: map[string]any{}}}
	send := func(claims map[string]any) int {
		form := url.Values{"logout_token": {signedJWT(t, f.key, claims)}}
		r := httptest.NewRequest("POST", "https://workspace.example/api/v1/auth/backchannel-logout", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		b.BackchannelLogout(w, r)
		return w.Code
	}
	for _, field := range []string{"iss", "aud", "events", "nonce", "iat"} {
		invalid := map[string]any{}
		for k, v := range claims {
			invalid[k] = v
		}
		invalid[field] = "wrong"
		if status := send(invalid); status != 400 {
			t.Fatalf("accepted invalid logout %s: %d", field, status)
		}
	}
	if status := send(claims); status != 200 {
		t.Fatalf("valid logout rejected: %d", status)
	}
	if status := send(claims); status != 400 {
		t.Fatal("logout replay accepted")
	}
	r := httptest.NewRequest("GET", "/api/v1/me", nil)
	r.AddCookie(cookie)
	if _, err := b.Authenticate(r); err == nil {
		t.Fatal("revoked session accepted")
	}
}
