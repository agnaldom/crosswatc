package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/correlation"
)

// Problem is an RFC 9457 problem document (spec §39).
type Problem struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	Status        int    `json:"status"`
	Detail        string `json:"detail,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

const problemBase = "https://crosswatc.io/errors/"

func problem(c *gin.Context, status int, slug, title, detail string) {
	id, _ := correlation.FromContext(c.Request.Context())
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, Problem{Type: problemBase + slug, Title: title, Status: status, Detail: detail, CorrelationID: id})
}

func notFound(c *gin.Context) {
	problem(c, http.StatusNotFound, "not-found", "Resource not found", "")
}

func internalError(c *gin.Context, err error) {
	_ = c.Error(err)
	problem(c, http.StatusInternalServerError, "internal", "Internal error", "")
}
