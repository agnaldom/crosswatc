package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/events"
)

// maxWebhookBody matches GitHub's 25 MB payload cap.
const maxWebhookBody = 25 << 20

// Inbox queues webhook deliveries for the worker (satisfied by *store.Store).
type Inbox interface {
	InsertInbox(ctx context.Context, source, deliveryID, eventType string, payload json.RawMessage) (bool, error)
}

// WebhookConfig holds the shared secrets of the webhook endpoints (spec §35.1). An empty secret disables
// the corresponding endpoint (404).
type WebhookConfig struct {
	GitHubSecret string // GitHub App webhook secret (X-Hub-Signature-256)
	ArgoCDToken  string // bearer token configured in argocd-notifications
}

type webhooks struct {
	inbox Inbox
	cfg   WebhookConfig
}

func (w *webhooks) register(r *gin.Engine) {
	g := r.Group("/webhooks")
	g.POST("/github", w.github)
	g.POST("/argocd", w.argocd)
}

func readBody(c *gin.Context) ([]byte, bool) {
	b, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBody))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			problem(c, http.StatusRequestEntityTooLarge, "payload-too-large", "Payload too large", "")
		} else {
			problem(c, http.StatusBadRequest, "invalid-body", "Invalid body", "")
		}
		return nil, false
	}
	return b, true
}

// validGitHubSignature checks X-Hub-Signature-256 = "sha256=" + hex(HMAC-SHA256(secret, body)) in constant time.
func validGitHubSignature(secret string, body []byte, header string) bool {
	sig, ok := strings.CutPrefix(header, "sha256=")
	if !ok {
		return false
	}
	got, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}

// github handles POST /webhooks/github. Invalid signature → 401 and nothing is persisted; duplicates
// (same X-GitHub-Delivery) → 202 without reprocessing (§35.1).
func (w *webhooks) github(c *gin.Context) {
	if w.cfg.GitHubSecret == "" {
		notFound(c)
		return
	}
	body, ok := readBody(c)
	if !ok {
		return
	}
	if !validGitHubSignature(w.cfg.GitHubSecret, body, c.GetHeader("X-Hub-Signature-256")) {
		problem(c, http.StatusUnauthorized, "invalid-signature", "Invalid signature", "")
		return
	}
	event, delivery := c.GetHeader("X-GitHub-Event"), c.GetHeader("X-GitHub-Delivery")
	if delivery == "" || event == "" {
		problem(c, http.StatusBadRequest, "missing-headers", "Missing headers", "X-GitHub-Event and X-GitHub-Delivery are required")
		return
	}
	if event == "ping" {
		c.JSON(http.StatusOK, gin.H{"status": "pong"})
		return
	}
	if !events.GitHubEvents[event] {
		c.JSON(http.StatusAccepted, gin.H{"status": "ignored"})
		return
	}
	if !json.Valid(body) {
		problem(c, http.StatusBadRequest, "invalid-body", "Invalid body", "payload must be JSON")
		return
	}
	w.enqueue(c, events.SourceGitHub, delivery, event, body)
}

// argocd handles POST /webhooks/argocd (argocd-notifications webhook service with Authorization: Bearer).
func (w *webhooks) argocd(c *gin.Context) {
	if w.cfg.ArgoCDToken == "" {
		notFound(c)
		return
	}
	tok, _ := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(tok), []byte(w.cfg.ArgoCDToken)) != 1 {
		problem(c, http.StatusUnauthorized, "invalid-token", "Invalid token", "")
		return
	}
	body, ok := readBody(c)
	if !ok {
		return
	}
	p, err := events.ParseArgo(body)
	if err != nil {
		problem(c, http.StatusBadRequest, "invalid-body", "Invalid body", err.Error())
		return
	}
	w.enqueue(c, events.SourceArgoCD, p.EventID, p.Event, body)
}

func (w *webhooks) enqueue(c *gin.Context, source, delivery, event string, body []byte) {
	inserted, err := w.inbox.InsertInbox(c.Request.Context(), source, delivery, event, body)
	if err != nil {
		internalError(c, err)
		return
	}
	status := "queued"
	if !inserted {
		status = "duplicate"
	}
	c.JSON(http.StatusAccepted, gin.H{"status": status})
}
