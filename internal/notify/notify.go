// Package notify delivers lifecycle notifications to Microsoft Teams and e-mail (spec §60).
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/agnaldom/crosswatc/internal/metrics"
	"github.com/agnaldom/crosswatc/internal/store"
)

// Kinds (spec §60 Notifications).
const (
	KindWaitingApproval = "WAITING_APPROVAL"
	KindFailed          = "FAILED"
	KindOperational     = "OPERATIONAL"
	KindDriftDetected   = "DRIFT_DETECTED"
)

// Route lists recipients for a squad.
type Route struct {
	Teams []string `json:"teams,omitempty"` // Teams Workflows/Incoming webhook URLs
	Email []string `json:"email,omitempty"`
	Kinds []string `json:"kinds,omitempty"` // empty: all kinds
}

// Config is the routing file (CROSSWATC_NOTIFY_CONFIG). It contains webhook URLs: keep it in a Secret.
//
//	default: {email: [plataforma@corp.com], kinds: [FAILED]}
//	squads:
//	  payments: {teams: ["https://…"], email: [payments@corp.com]}
type Config struct {
	Default *Route           `json:"default,omitempty"`
	Squads  map[string]Route `json:"squads,omitempty"`
}

// LoadConfig reads a YAML/JSON routing file.
func LoadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path) //nolint:gosec // operator-provided config path
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("notify: %s: %w", path, err)
	}
	return c, nil
}

// routes returns every route that applies to (owner, kind): the squad's and the default.
func (c Config) routes(owner, kind string) []Route {
	var out []Route
	if r, ok := c.Squads[owner]; ok && wants(r, kind) {
		out = append(out, r)
	}
	if c.Default != nil && wants(*c.Default, kind) {
		out = append(out, *c.Default)
	}
	return out
}

func wants(r Route, kind string) bool { return len(r.Kinds) == 0 || slices.Contains(r.Kinds, kind) }

// Message is what a channel renders.
type Message struct {
	Kind          string    `json:"-"`
	Owner         string    `json:"-"`
	EventType     string    `json:"eventType"`
	Text          string    `json:"message"`
	ResourceName  string    `json:"resourceName"`
	Environment   string    `json:"environment"`
	ResourceKind  string    `json:"kind"`
	Provider      string    `json:"provider"`
	PRURL         string    `json:"prUrl"`
	CorrelationID string    `json:"correlationId"`
	OccurredAt    time.Time `json:"occurredAt"`
	Link          string    `json:"-"`
}

var titles = map[string]string{
	KindWaitingApproval: "Aguardando aprovação",
	KindFailed:          "Falha",
	KindOperational:     "Recurso operacional",
	KindDriftDetected:   "Drift de configuração detectado",
}

// Title is the human headline of the message.
func (m Message) Title() string {
	t := titles[m.Kind]
	if t == "" {
		t = m.Kind
	}
	return fmt.Sprintf("%s — %s (%s)", t, m.ResourceName, m.Environment)
}

// Sender delivers to one channel.
type Sender interface {
	Channel() string
	Send(ctx context.Context, to []string, m Message) error
}

// Dispatcher turns outbox rows into channel deliveries.
type Dispatcher struct {
	Config    Config
	Teams     Sender // optional
	Email     Sender // optional
	PortalURL string // e.g. https://crosswatc.corp.com
}

// Handle implements the store.ProcessNotifications callback.
func (d *Dispatcher) Handle(ctx context.Context, n store.Notification) error {
	var m Message
	if err := json.Unmarshal(n.Payload, &m); err != nil {
		return fmt.Errorf("%w: payload: %w", store.ErrPermanent, err)
	}
	m.Kind, m.Owner = n.Kind, n.Owner
	if d.PortalURL != "" && n.ResourceID != nil {
		m.Link = strings.TrimRight(d.PortalURL, "/") + "/infrastructure/" + n.ResourceID.String()
	}
	routes := d.Config.routes(n.Owner, n.Kind)
	var teams, email []string
	for _, r := range routes {
		teams = append(teams, r.Teams...)
		email = append(email, r.Email...)
	}
	teams, email = dedupe(teams), dedupe(email)
	sent := false
	var errs []error
	for _, ch := range []struct {
		s  Sender
		to []string
	}{{d.Teams, teams}, {d.Email, email}} {
		if ch.s == nil || len(ch.to) == 0 {
			continue
		}
		err := ch.s.Send(ctx, ch.to, m)
		status := "sent"
		if err != nil {
			status = "error"
			errs = append(errs, fmt.Errorf("%s: %w", ch.s.Channel(), err))
		} else {
			sent = true
		}
		metrics.NotificationsTotal.WithLabelValues(ch.s.Channel(), n.Kind, status).Inc()
	}
	if len(errs) > 0 && !sent {
		return errors.Join(errs...)
	}
	if !sent && len(errs) == 0 {
		metrics.NotificationsTotal.WithLabelValues("none", n.Kind, "skipped").Inc()
		return fmt.Errorf("%w: no recipients for squad %q", store.ErrSkip, n.Owner)
	}
	return nil // partial success is not retried to avoid duplicate deliveries on the healthy channel
}

func dedupe(in []string) []string {
	slices.Sort(in)
	return slices.Compact(in)
}
