package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/forms"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/store"
)

// formFor builds the form of a kind (claim kind or XR kind).
func (h *handlers) formFor(c *gin.Context, kind string) (forms.Form, kube.XRD, bool) {
	xrds, ok := h.xrds(c)
	if !ok {
		return forms.Form{}, kube.XRD{}, false
	}
	x, found := findKind(xrds, kind)
	if !found {
		notFound(c)
		return forms.Form{}, kube.XRD{}, false
	}
	f, err := forms.Build(x)
	if err != nil {
		problem(c, http.StatusUnprocessableEntity, "invalid-ui-metadata", "Invalid UI metadata on XRD", err.Error())
		return forms.Form{}, kube.XRD{}, false
	}
	return f, x, true
}

func (h *handlers) getResourceTypeForm(c *gin.Context) {
	if f, _, ok := h.formFor(c, c.Param("kind")); ok {
		c.JSON(http.StatusOK, f)
	}
}

type resourceFormDTO struct {
	Form     forms.Form     `json:"form"`
	Values   map[string]any `json:"values"`
	Editable bool           `json:"editable"`
	Reason   string         `json:"reason,omitempty"`
}

// appliedSpec reads the Claim spec recorded by the worker (spec §18 AppliedState).
func (h *handlers) appliedSpec(c *gin.Context, r store.TrackedResource) (map[string]any, bool) {
	p, err := h.store.GetProjection(c.Request.Context(), r.ID)
	if errors.Is(err, store.ErrNotFound) {
		return map[string]any{}, true
	}
	if err != nil {
		internalError(c, err)
		return nil, false
	}
	var applied struct {
		Spec map[string]any `json:"spec"`
	}
	if len(p.AppliedState) > 0 {
		if err := json.Unmarshal(p.AppliedState, &applied); err != nil {
			internalError(c, err)
			return nil, false
		}
	}
	if applied.Spec == nil {
		applied.Spec = map[string]any{}
	}
	return applied.Spec, true
}

func (h *handlers) getResourceForm(c *gin.Context) {
	r := resourceOf(c)
	f, _, ok := h.formFor(c, r.Kind)
	if !ok {
		return
	}
	spec, ok := h.appliedSpec(c, r)
	if !ok {
		return
	}
	out := resourceFormDTO{Form: f, Values: forms.Values(f, spec), Editable: r.ResourceState == "OPERATIONAL"}
	if !out.Editable {
		out.Reason = "Somente recursos OPERATIONAL podem ser editados (estado atual: " + r.ResourceState + ")"
	}
	c.JSON(http.StatusOK, out)
}
