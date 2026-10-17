// Package auth implements AuthN (OIDC or dev headers) and squad-based AuthZ (spec §49).
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// ErrUnauthenticated means no valid identity was presented.
var ErrUnauthenticated = errors.New("auth: unauthenticated")

// Principal is the authenticated caller.
type Principal struct {
	User   string
	Squads []string
	SRE    bool // global read
}

// CanRead reports resource.owner ∈ user.squads, or SRE.
func (p Principal) CanRead(owner string) bool {
	if p.SRE {
		return true
	}
	for _, s := range p.Squads {
		if s == owner {
			return true
		}
	}
	return false
}

// OwnerScope returns the owner filter for listings: nil means unrestricted (SRE).
func (p Principal) OwnerScope() []string {
	if p.SRE {
		return nil
	}
	if p.Squads == nil {
		return []string{}
	}
	return p.Squads
}

// Authenticator resolves the caller of an HTTP request.
type Authenticator interface {
	Authenticate(r *http.Request) (Principal, error)
}

// GroupMapping converts IdP groups to squads (spec §49).
type GroupMapping struct {
	SquadPrefix string // e.g. "squad-" → "squad-payments" becomes "payments"
	SREGroup    string
}

func (m GroupMapping) apply(user string, groups []string) Principal {
	p := Principal{User: user}
	for _, g := range groups {
		if g == m.SREGroup && m.SREGroup != "" {
			p.SRE = true
		}
		if s, ok := strings.CutPrefix(g, m.SquadPrefix); ok && s != "" && m.SquadPrefix != "" {
			p.Squads = append(p.Squads, s)
		}
	}
	return p
}

// Dev authenticates from X-Dev-User / X-Dev-Squads headers. Never enable in production (enforced by config).
type Dev struct{ Mapping GroupMapping }

func (d Dev) Authenticate(r *http.Request) (Principal, error) {
	user := r.Header.Get("X-Dev-User")
	if user == "" {
		return Principal{}, ErrUnauthenticated
	}
	var groups []string
	for _, s := range strings.Split(r.Header.Get("X-Dev-Squads"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			groups = append(groups, d.Mapping.SquadPrefix+s)
		}
	}
	if strings.EqualFold(r.Header.Get("X-Dev-SRE"), "true") {
		groups = append(groups, d.Mapping.SREGroup)
	}
	return d.Mapping.apply(user, groups), nil
}

// OIDC verifies bearer ID/access tokens (JWT) against the issuer's JWKS.
type OIDC struct {
	verifier *oidc.IDTokenVerifier
	mapping  GroupMapping
}

// NewOIDC discovers the issuer configuration.
func NewOIDC(ctx context.Context, issuer, audience string, m GroupMapping) (*OIDC, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("auth: oidc discovery: %w", err)
	}
	return &OIDC{verifier: provider.Verifier(&oidc.Config{ClientID: audience}), mapping: m}, nil
}

// NewOIDCWithVerifier builds an authenticator from an existing verifier (tests).
func NewOIDCWithVerifier(v *oidc.IDTokenVerifier, m GroupMapping) *OIDC {
	return &OIDC{verifier: v, mapping: m}
}

func (o *OIDC) Authenticate(r *http.Request) (Principal, error) {
	raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || raw == "" {
		return Principal{}, ErrUnauthenticated
	}
	tok, err := o.verifier.Verify(r.Context(), raw)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %w", ErrUnauthenticated, err)
	}
	var claims struct {
		PreferredUsername string   `json:"preferred_username"`
		Email             string   `json:"email"`
		Groups            []string `json:"groups"`
	}
	if err := tok.Claims(&claims); err != nil {
		return Principal{}, fmt.Errorf("%w: %w", ErrUnauthenticated, err)
	}
	user := claims.PreferredUsername
	if user == "" {
		user = claims.Email
	}
	if user == "" {
		user = tok.Subject
	}
	return o.mapping.apply(user, claims.Groups), nil
}
