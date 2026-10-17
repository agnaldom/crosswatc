package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/stream"
)

// Streamer is the realtime fan-out (satisfied by *stream.Hub).
type Streamer interface {
	Subscribe(f stream.Filter) *stream.Subscription
	Unsubscribe(s *stream.Subscription)
	Replay(ctx context.Context, lastID int64, f stream.Filter) ([]store.StreamEvent, error)
}

// DefaultHeartbeat is the SSE keep-alive interval (§45).
const DefaultHeartbeat = 20 * time.Second

// streamFilter applies the read AuthZ of §49 (owner ∈ squads, or SRE) and the optional resource filter.
// Events without an owner (resource unknown) are only visible to SRE.
func streamFilter(p auth.Principal, only *uuid.UUID) stream.Filter {
	return func(e store.StreamEvent) bool {
		if only != nil && (e.ResourceID == nil || *e.ResourceID != *only) {
			return false
		}
		if e.Owner == "" {
			return p.SRE
		}
		return p.CanRead(e.Owner)
	}
}

func writeEvent(w io.Writer, e store.StreamEvent) error {
	var data bytes.Buffer
	if err := json.Compact(&data, e.Data); err != nil {
		data.Reset()
		data.WriteString("{}")
	}
	_, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", e.ID, e.Type, data.Bytes())
	return err
}

// stream serves GET /api/v1/stream (§45): SSE with AuthZ filtering, ?resourceId, Last-Event-ID replay of the
// buffered window and a heartbeat comment.
func (h *handlers) stream(c *gin.Context) {
	if h.streamer == nil {
		problem(c, http.StatusServiceUnavailable, "stream-unavailable", "Stream unavailable", "")
		return
	}
	p := principal(c)
	var only *uuid.UUID
	if q := c.Query("resourceId"); q != "" {
		id, err := uuid.Parse(q)
		if err != nil {
			notFound(c)
			return
		}
		r, err := h.store.GetResource(c.Request.Context(), id)
		switch {
		case errors.Is(err, store.ErrNotFound) || (err == nil && !p.CanRead(r.Owner)):
			notFound(c) // do not leak existence
			return
		case err != nil:
			internalError(c, err)
			return
		}
		only = &id
	}
	lastRaw := c.GetHeader("Last-Event-ID")
	if lastRaw == "" {
		lastRaw = c.Query("lastEventId") // EventSource polyfills / manual resume
	}
	var lastID int64
	if lastRaw != "" {
		n, err := strconv.ParseInt(lastRaw, 10, 64)
		if err != nil || n < 0 {
			problem(c, http.StatusBadRequest, "invalid-last-event-id", "Invalid Last-Event-ID", "")
			return
		}
		lastID = n
	}

	filter := streamFilter(p, only)
	sub := h.streamer.Subscribe(filter) // subscribe before replaying so nothing falls in between
	defer h.streamer.Unsubscribe(sub)

	ctx := c.Request.Context()
	var replayed map[int64]bool
	var backlog []store.StreamEvent
	if lastRaw != "" {
		evs, err := h.streamer.Replay(ctx, lastID, filter)
		if err != nil {
			internalError(c, err)
			return
		}
		backlog = evs
		replayed = make(map[int64]bool, len(evs))
		for _, e := range evs {
			replayed[e.ID] = true
		}
	}

	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(w, "retry: 3000\n\n"); err != nil {
		return
	}
	for _, e := range backlog {
		if writeEvent(w, e) != nil {
			return
		}
	}
	w.Flush()

	hb := h.heartbeat
	if hb <= 0 {
		hb = DefaultHeartbeat
	}
	ticker := time.NewTicker(hb)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
		case e, ok := <-sub.C:
			if !ok {
				return // dropped as slow or hub stopping: the client resumes with Last-Event-ID
			}
			if replayed[e.ID] {
				continue
			}
			if writeEvent(w, e) != nil {
				return
			}
		}
		w.Flush()
	}
}
