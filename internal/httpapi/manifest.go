package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/kube"
)

type manifestObject struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// GET /resources/{id}/manifest?object=<topology node id> — sanitized live YAML of the Claim (default), XR or a MR.
// Only objects in the resource's topology can be requested.
func (h *handlers) getManifest(c *gin.Context) {
	if h.cluster == nil {
		problem(c, http.StatusServiceUnavailable, "cluster-unavailable", "Kubernetes access not configured", "")
		return
	}
	refs, nodes, ok := h.topologyRefs(c, resourceOf(c))
	if !ok {
		return
	}
	if len(refs) == 0 {
		problem(c, http.StatusNotFound, "no-topology", "Topology not known yet", "")
		return
	}
	objects := make([]manifestObject, len(nodes))
	pick := 0
	want := c.Query("object")
	found := want == ""
	for i, n := range nodes {
		objects[i] = manifestObject{ID: n.ID, Type: n.Type, Kind: n.Kind, Name: n.Name, Namespace: n.Namespace}
		if want != "" && n.ID == want {
			pick, found = i, true
		}
	}
	if !found {
		problem(c, http.StatusNotFound, "object-not-in-topology", "Object is not part of this resource", "")
		return
	}
	y, err := h.cluster.Manifest(c.Request.Context(), refs[pick])
	switch {
	case errors.Is(err, kube.ErrForbiddenKind):
		problem(c, http.StatusForbidden, "kind-not-viewable", "This kind cannot be displayed", "")
		return
	case kube.IsNotFound(err):
		problem(c, http.StatusNotFound, "object-gone", "Object no longer exists in the cluster", "")
		return
	case err != nil:
		_ = c.Error(err)
		problem(c, http.StatusBadGateway, "cluster-error", "Could not read the object", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": objects[pick], "yaml": string(y), "objects": objects})
}
