package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Teams posts Adaptive Cards to Teams Workflows / Incoming webhooks.
type Teams struct {
	HTTP *http.Client
}

func (Teams) Channel() string { return "teams" }

func (t Teams) Send(ctx context.Context, to []string, m Message) error {
	body, err := json.Marshal(card(m))
	if err != nil {
		return err
	}
	hc := t.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	for _, url := range to {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := hc.Do(req)
		if err != nil {
			return fmt.Errorf("teams webhook: %w", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("teams webhook: HTTP %d", resp.StatusCode)
		}
	}
	return nil
}

var colors = map[string]string{KindFailed: "Attention", KindDriftDetected: "Warning", KindOperational: "Good", KindWaitingApproval: "Accent"}

func card(m Message) map[string]any {
	facts := []map[string]string{
		{"title": "Recurso", "value": m.ResourceName},
		{"title": "Tipo", "value": m.ResourceKind},
		{"title": "Ambiente", "value": m.Environment},
		{"title": "Squad", "value": m.Owner},
		{"title": "Correlation ID", "value": m.CorrelationID},
	}
	var actions []map[string]string
	if m.Link != "" {
		actions = append(actions, map[string]string{"type": "Action.OpenUrl", "title": "Abrir no Crosswatc", "url": m.Link})
	}
	if m.PRURL != "" {
		actions = append(actions, map[string]string{"type": "Action.OpenUrl", "title": "Pull Request", "url": m.PRURL})
	}
	content := map[string]any{
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"type":    "AdaptiveCard",
		"version": "1.4",
		"body": []any{
			map[string]any{"type": "TextBlock", "size": "Medium", "weight": "Bolder", "wrap": true, "text": m.Title(), "color": colors[m.Kind]},
			map[string]any{"type": "TextBlock", "wrap": true, "text": m.Text},
			map[string]any{"type": "FactSet", "facts": facts},
		},
		"actions": actions,
	}
	return map[string]any{
		"type":        "message",
		"attachments": []any{map[string]any{"contentType": "application/vnd.microsoft.card.adaptive", "content": content}},
	}
}
