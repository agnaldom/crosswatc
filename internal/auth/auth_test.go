package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

var mapping = GroupMapping{SquadPrefix: "squad-", SREGroup: "platform-sre"}

func TestPrincipal(t *testing.T) {
	p := Principal{Squads: []string{"payments"}}
	if !p.CanRead("payments") || p.CanRead("cards") || len(p.OwnerScope()) != 1 {
		t.Fatal("squad member")
	}
	if !p.CanWrite("payments") || p.CanWrite("cards") || (Principal{SRE: true}).CanWrite("payments") {
		t.Fatal("write")
	}
	if !(Principal{SRE: true}).CanRead("x") || (Principal{SRE: true}).OwnerScope() != nil {
		t.Fatal("sre")
	}
	if s := (Principal{}).OwnerScope(); s == nil || len(s) != 0 {
		t.Fatal("no squads must yield empty, non-nil scope")
	}
}

func TestDev(t *testing.T) {
	d := Dev{Mapping: mapping}
	r := httptest.NewRequest("GET", "/", nil)
	if _, err := d.Authenticate(r); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("missing user must fail")
	}
	r.Header.Set("X-Dev-User", "agnaldo")
	r.Header.Set("X-Dev-Squads", "payments, cards")
	p, err := d.Authenticate(r)
	if err != nil || p.User != "agnaldo" || len(p.Squads) != 2 || p.SRE {
		t.Fatalf("%+v %v", p, err)
	}
	r.Header.Set("X-Dev-SRE", "true")
	if p, _ := d.Authenticate(r); !p.SRE {
		t.Fatal("sre header")
	}
}

// signedJWT builds an RS256 JWT for tests.
func signedJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
	head := enc(map[string]any{"alg": "RS256", "typ": "JWT", "kid": "k1"})
	body := enc(claims)
	h := sha256.Sum256([]byte(head + "." + body))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		t.Fatal(err)
	}
	return head + "." + body + "." + base64.RawURLEncoding.EncodeToString(sig)
}

type staticKeys struct{ key *rsa.PublicKey }

func (s staticKeys) VerifySignature(_ context.Context, jwt string) ([]byte, error) {
	// delegate to go-oidc's StaticKeySet
	return (&oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{s.key}}).VerifySignature(context.Background(), jwt)
}

func TestOIDC(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := oidc.NewVerifier("https://idp.example.com", staticKeys{&key.PublicKey}, &oidc.Config{ClientID: "crosswatc"})
	o := NewOIDCWithVerifier(v, mapping)

	tok := signedJWT(t, key, map[string]any{
		"iss": "https://idp.example.com", "aud": "crosswatc", "sub": "u1", "exp": time.Now().Add(time.Hour).Unix(),
		"preferred_username": "agnaldo", "groups": []string{"squad-payments", "platform-sre", "other"},
	})
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	p, err := o.Authenticate(r)
	if err != nil || p.User != "agnaldo" || !p.SRE || len(p.Squads) != 1 || p.Squads[0] != "payments" {
		t.Fatalf("%+v %v", p, err)
	}

	bad := signedJWT(t, key, map[string]any{"iss": "https://idp.example.com", "aud": "other", "sub": "u1", "exp": time.Now().Add(time.Hour).Unix()})
	r.Header.Set("Authorization", "Bearer "+bad)
	if _, err := o.Authenticate(r); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("wrong audience must fail")
	}
	r.Header.Del("Authorization")
	if _, err := o.Authenticate(r); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("missing token must fail")
	}
	roles := NewOIDCWithVerifier(v, GroupMapping{SquadPrefix: "squad-", SREGroup: "platform-sre", GroupsClaim: "roles"})
	rt := signedJWT(t, key, map[string]any{"iss": "https://idp.example.com", "aud": "crosswatc", "sub": "u3",
		"exp": time.Now().Add(time.Hour).Unix(), "roles": "squad-cards platform-sre"})
	r.Header.Set("Authorization", "Bearer "+rt)
	if p, err := roles.Authenticate(r); err != nil || !p.SRE || p.Squads[0] != "cards" {
		t.Fatalf("roles claim: %+v %v", p, err)
	}
	sub := signedJWT(t, key, map[string]any{"iss": "https://idp.example.com", "aud": "crosswatc", "sub": "u2", "exp": time.Now().Add(time.Hour).Unix()})
	r.Header.Set("Authorization", "Bearer "+sub)
	if p, err := o.Authenticate(r); err != nil || p.User != "u2" {
		t.Fatalf("subject fallback: %+v %v", p, err)
	}
}
