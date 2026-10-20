package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/kube"
)

// Platform reads Crossplane installation health (satisfied by *kube.Client).
type Platform interface {
	Packages(ctx context.Context, kind string) ([]kube.Package, error)
	Compositions(ctx context.Context) ([]kube.Composition, error)
	DiscoverXRDs(ctx context.Context) ([]kube.XRD, error)
	MRDs(ctx context.Context) ([]kube.MRD, error)
	MRAPs(ctx context.Context) ([]kube.MRAP, error)
}

// GET /api/v1/me — lets the UI adapt to the caller's profile.
func (h *handlers) me(c *gin.Context) {
	p := principal(c)
	squads := p.Squads
	if squads == nil {
		squads = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"user": p.User, "squads": squads, "sre": p.SRE})
}

// requireSRE guards the platform catalog: it exposes the whole Crossplane installation.
func requireSRE(c *gin.Context) {
	if !principal(c).SRE {
		problem(c, http.StatusForbidden, "sre-only", "Platform catalog is restricted to SRE/platform", "")
		return
	}
	c.Next()
}

func (h *handlers) platformAvailable(c *gin.Context) bool {
	if h.platform == nil {
		problem(c, http.StatusServiceUnavailable, "cluster-unavailable", "Kubernetes access not configured", "")
		return false
	}
	return true
}

func (h *handlers) listPackages(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.platformAvailable(c) {
			return
		}
		ps, err := h.platform.Packages(c.Request.Context(), kind)
		if err != nil {
			_ = c.Error(err)
			problem(c, http.StatusBadGateway, "cluster-error", "Could not list "+kind+"s", err.Error())
			return
		}
		c.JSON(http.StatusOK, newPage(ps, ""))
	}
}

func (h *handlers) listCompositions(c *gin.Context) {
	if !h.platformAvailable(c) {
		return
	}
	cs, err := h.platform.Compositions(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		problem(c, http.StatusBadGateway, "cluster-error", "Could not list compositions", err.Error())
		return
	}
	c.JSON(http.StatusOK, newPage(cs, ""))
}

type platformXRD struct {
	Name        string `json:"name"`
	Group       string `json:"group"`
	Kind        string `json:"kind"`
	ClaimKind   string `json:"claimKind,omitempty"`
	Scope       string `json:"scope"`
	Version     string `json:"version"`
	Established bool   `json:"established"`
}

func (h *handlers) listPlatformXRDs(c *gin.Context) {
	if !h.platformAvailable(c) {
		return
	}
	xrds, err := h.platform.DiscoverXRDs(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		problem(c, http.StatusBadGateway, "cluster-error", "Could not list XRDs", err.Error())
		return
	}
	out := make([]platformXRD, len(xrds))
	for i, x := range xrds {
		out[i] = platformXRD{Name: x.Name, Group: x.Group, Kind: x.Kind, ClaimKind: x.ClaimKind, Scope: x.Scope, Version: x.Version(), Established: x.Established}
	}
	c.JSON(http.StatusOK, newPage(out, ""))
}

// v2List serves a Crossplane v2 list; on a v1 cluster (API not served) it answers an empty page with
// available=false instead of an error, so the catalog degrades gracefully.
func v2List[T any](h *handlers, what string, list func(*gin.Context) ([]T, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.platformAvailable(c) {
			return
		}
		items, err := list(c)
		if kube.IsMissingAPI(err) {
			c.JSON(http.StatusOK, gin.H{"items": []T{}, "available": false})
			return
		}
		if err != nil {
			_ = c.Error(err)
			problem(c, http.StatusBadGateway, "cluster-error", "Could not list "+what, err.Error())
			return
		}
		if items == nil {
			items = []T{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "available": true})
	}
}

func (h *handlers) listMRDs() gin.HandlerFunc {
	return v2List(h, "managed resource definitions", func(c *gin.Context) ([]kube.MRD, error) {
		return h.platform.MRDs(c.Request.Context())
	})
}

func (h *handlers) listMRAPs() gin.HandlerFunc {
	return v2List(h, "activation policies", func(c *gin.Context) ([]kube.MRAP, error) {
		return h.platform.MRAPs(c.Request.Context())
	})
}
