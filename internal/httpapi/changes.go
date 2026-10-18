package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/store"
)

const maxChangeBody = 64 << 10

type changeRequest struct {
	Changes       map[string]any `json:"changes"`
	Justification string         `json:"justification"`
}

// requireWrite enforces resource.owner ∈ user.squads for mutations; SRE read is not enough.
func requireWrite(c *gin.Context, r store.TrackedResource) bool {
	if principal(c).CanWrite(r.Owner) {
		return true
	}
	problem(c, http.StatusForbidden, "forbidden", "Not allowed to change this resource",
		"Only members of squad "+r.Owner+" can request changes")
	return false
}

func bindChange(c *gin.Context) (changeRequest, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChangeBody)
	var req changeRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		problem(c, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return req, false
	}
	return req, true
}

// previewInput loads form + applied spec for a change against r.
func (h *handlers) previewInput(c *gin.Context, r store.TrackedResource, req changeRequest) (changes.Input, bool) {
	f, _, ok := h.formFor(c, r.Kind)
	if !ok {
		return changes.Input{}, false
	}
	spec, ok := h.appliedSpec(c, r)
	if !ok {
		return changes.Input{}, false
	}
	return changes.Input{
		Form: f, CurrentSpec: spec, Changes: req.Changes, Environment: r.Environment, ResourceState: r.ResourceState,
		ResourceName: r.Name, Owner: r.Owner, Provider: r.Provider, Kind: r.Kind,
	}, true
}

func (h *handlers) previewChanges(c *gin.Context) {
	r := resourceOf(c)
	if !requireWrite(c, r) {
		return
	}
	req, ok := bindChange(c)
	if !ok {
		return
	}
	in, ok := h.previewInput(c, r, req)
	if !ok {
		return
	}
	res, err := changes.Preview(c.Request.Context(), in, h.policies...)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
