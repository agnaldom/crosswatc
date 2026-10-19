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

// Handle implements the store.ProcessNotifications callback. Each recipient is a separate target
// ("teams:<url>", "email:<addr>"); targets already in n.SentTargets are skipped, so a retry only reaches
// the ones that failed. It returns every target delivered in this call, even when others failed.
func (d *Dispatcher) Handle(ctx context.Context, n store.Notification) ([]string, error) {
	var m Message
	if err := json.Unmarshal(n.Payload, &m); err != nil {
		return nil, fmt.Errorf("%w: payload: %w", store.ErrPermanent, err)
	}
	m.Kind, m.Owner = n.Kind, n.Owner
	if d.PortalURL != "" && n.ResourceID != nil {
		m.Link = strings.TrimRight(d.PortalURL, "/") + "/infrastructure/" + n.ResourceID.String()
	}
	type target struct {
		key  string
		send Sender
		to   string
	}
	var targets []target
	for _, r := range d.Config.routes(n.Owner, n.Kind) {
		if d.Teams != nil {
			for _, u := range r.Teams {
				targets = append(targets, target{"teams:" + u, d.Teams, u})
			}
		}
		if d.Email != nil {
			for _, a := range r.Email {
				targets = append(targets, target{"email:" + a, d.Email, a})
			}
		}
	}
	if len(targets) == 0 {
		metrics.NotificationsTotal.WithLabelValues("none", n.Kind, "skipped").Inc()
		return nil, fmt.Errorf("%w: no recipients for squad %q", store.ErrSkip, n.Owner)
	}
	done := map[string]bool{}
	for _, t := range n.SentTargets {
		done[t] = true
	}
	var delivered []string
	var errs []error
	for _, t := range targets {
		if done[t.key] {
			continue
		}
		done[t.key] = true // routes may repeat a recipient (squad + default)
		if err := t.send.Send(ctx, []string{t.to}, m); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", t.send.Channel(), err))
			metrics.NotificationsTotal.WithLabelValues(t.send.Channel(), n.Kind, "error").Inc()
			continue
		}
		delivered = append(delivered, t.key)
		metrics.NotificationsTotal.WithLabelValues(t.send.Channel(), n.Kind, "sent").Inc()
	}
	return delivered, errors.Join(errs...)
}
