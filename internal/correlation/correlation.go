// Package correlation generates and validates Crosswatc correlation IDs (spec §16).
package correlation

import (
	"context"
	"crypto/rand"
	"regexp"
	"time"

	"github.com/oklog/ulid/v2"
)

// Prefix is prepended to every correlation ID.
const Prefix = "crosswatc-req-"

// Header is the HTTP header carrying the correlation ID.
const Header = "X-Correlation-Id"

var pattern = regexp.MustCompile(`^crosswatc-req-[0-9A-HJKMNP-TV-Z]{26}$`)

// New returns a fresh ID using the current time.
func New() string { return NewAt(time.Now()) }

// NewAt returns a fresh ID whose ULID timestamp is t.
func NewAt(t time.Time) string {
	return Prefix + ulid.MustNew(ulid.Timestamp(t), rand.Reader).String()
}

// Valid reports whether id matches the §16 format.
func Valid(id string) bool { return pattern.MatchString(id) }

// Time extracts the creation time encoded in id.
func Time(id string) (time.Time, bool) {
	if !Valid(id) {
		return time.Time{}, false
	}
	u, err := ulid.ParseStrict(id[len(Prefix):])
	if err != nil {
		return time.Time{}, false
	}
	return ulid.Time(u.Time()), true
}

type ctxKey struct{}

// WithID stores id in ctx.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the ID in ctx, if any.
func FromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok
}
