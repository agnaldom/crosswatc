package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type attentionDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	State  string `json:"resourceState"`
	Drift  bool   `json:"drift"`
	Synced *bool  `json:"synced,omitempty"`
	Ready  *bool  `json:"ready,omitempty"`
}

type openRevisionDTO struct {
	ResourceID   string     `json:"resourceId"`
	ResourceName string     `json:"resourceName"`
	Rev          int        `json:"rev"`
	Kind         string     `json:"kind"`
	State        string     `json:"state"`
	PRURL        string     `json:"prUrl,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
}

type dashboardEventDTO struct {
	eventDTO
	ResourceName string `json:"resourceName"`
}

type dashboardDTO struct {
	Total         int                 `json:"total"`
	ByState       map[string]int      `json:"byState"`
	Attention     []attentionDTO      `json:"attention"`
	OpenRevisions []openRevisionDTO   `json:"openRevisions"`
	RecentEvents  []dashboardEventDTO `json:"recentEvents"`
}

// dashboard serves the home page aggregates, scoped by squad like the resource list (§39).
func (h *handlers) dashboard(c *gin.Context) {
	d, err := h.store.GetDashboard(c.Request.Context(), principal(c).OwnerScope())
	if err != nil {
		internalError(c, err)
		return
	}
	out := dashboardDTO{ByState: d.ByState, Attention: []attentionDTO{}, OpenRevisions: []openRevisionDTO{},
		RecentEvents: []dashboardEventDTO{}}
	for _, n := range d.ByState {
		out.Total += n
	}
	for _, a := range d.Attention {
		out.Attention = append(out.Attention, attentionDTO{ID: a.ID.String(), Name: a.Name, Kind: a.Kind, State: a.State,
			Drift: a.Drift, Synced: a.Synced, Ready: a.Ready})
	}
	for _, r := range d.OpenRevisions {
		out.OpenRevisions = append(out.OpenRevisions, openRevisionDTO{ResourceID: r.ResourceID.String(),
			ResourceName: r.ResourceName, Rev: r.Rev, Kind: r.Kind, State: r.State, PRURL: r.PRURL, StartedAt: r.StartedAt})
	}
	for _, e := range d.RecentEvents {
		out.RecentEvents = append(out.RecentEvents, dashboardEventDTO{eventDTO: toEventDTO(e.Event), ResourceName: e.ResourceName})
	}
	c.JSON(http.StatusOK, out)
}
