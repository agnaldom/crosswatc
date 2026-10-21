package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *handlers) getContexts(c *gin.Context) {
	if h.contexts == nil {
		c.JSON(http.StatusOK, gin.H{
			"contexts":       []string{"in-cluster"},
			"currentContext": "in-cluster",
			"isInCluster":    true,
		})
		return
	}
	contexts, current, inCluster, err := h.contexts.GetContexts()
	if err != nil {
		problem(c, http.StatusInternalServerError, "contexts-failed", "Failed to get Kubernetes contexts", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"contexts":       contexts,
		"currentContext": current,
		"isInCluster":    inCluster,
	})
}

func (h *handlers) setCurrentContext(c *gin.Context) {
	var req struct {
		Context string `json:"context" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		problem(c, http.StatusBadRequest, "invalid-context", "context parameter is required", "")
		return
	}
	if h.contexts == nil {
		problem(c, http.StatusBadRequest, "in-cluster-mode", "Cannot switch context in in-cluster mode", "")
		return
	}
	if err := h.contexts.SetContext(req.Context); err != nil {
		problem(c, http.StatusBadRequest, "set-context-failed", "Failed to set context", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "context": req.Context})
}

func (h *handlers) checkContextConnection(c *gin.Context) {
	contextName := c.Query("context")
	if h.contexts == nil {
		c.JSON(http.StatusOK, gin.H{"connected": true, "context": "in-cluster"})
		return
	}
	connected, err := h.contexts.CheckConnection(contextName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"connected": false, "context": contextName, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connected": connected, "context": contextName})
}

func (h *handlers) addKubeConfig(c *gin.Context) {
	var req struct {
		KubeConfig string `json:"kubeConfig" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.KubeConfig == "" {
		problem(c, http.StatusBadRequest, "invalid-kubeconfig", "kubeConfig content is required", "")
		return
	}
	if h.contexts == nil {
		problem(c, http.StatusBadRequest, "in-cluster-mode", "Cannot add contexts in in-cluster mode", "")
		return
	}
	added, err := h.contexts.AddKubeConfig(req.KubeConfig)
	if err != nil {
		problem(c, http.StatusBadRequest, "add-context-failed", "Failed to add kubeconfig", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"addedContexts": added,
		"message":       "Context added successfully",
	})
}

func (h *handlers) removeContext(c *gin.Context) {
	var req struct {
		Context string `json:"context" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Context == "" {
		problem(c, http.StatusBadRequest, "invalid-context", "context parameter is required", "")
		return
	}
	if h.contexts == nil {
		problem(c, http.StatusBadRequest, "in-cluster-mode", "Cannot remove contexts in in-cluster mode", "")
		return
	}
	if err := h.contexts.RemoveContext(req.Context); err != nil {
		problem(c, http.StatusBadRequest, "remove-context-failed", "Failed to remove context", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Context removed successfully",
	})
}
