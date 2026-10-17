// Package contract defines the crosswatc.io labels/annotations contract (spec §17).
package contract

import (
	"fmt"
	"strconv"

	"github.com/agnaldom/crosswatc/internal/correlation"
)

const Domain = "crosswatc.io"

// Labels.
const (
	LabelManaged      = Domain + "/managed"
	LabelOwner        = Domain + "/owner"
	LabelEnvironment  = Domain + "/environment"
	LabelProvider     = Domain + "/provider"
	LabelResourceType = Domain + "/resource-type"
)

// Annotations.
const (
	AnnotationRequestID       = Domain + "/request-id"
	AnnotationRevision        = Domain + "/revision"
	AnnotationRepository      = Domain + "/repository"
	AnnotationPath            = Domain + "/path"
	AnnotationArgoApp         = Domain + "/argo-app"
	AnnotationBackstageTaskID = Domain + "/backstage-task-id"
)

// ManagedSelector selects Claims tracked by Crosswatc (spec §48).
const ManagedSelector = LabelManaged + "=true"

// Providers supported by TrackedResource.provider.
var Providers = map[string]bool{"azure": true, "aws": true, "huawei": true}

// Metadata is the parsed contract of a Claim.
type Metadata struct {
	Owner           string
	Environment     string
	Provider        string
	ResourceType    string
	RequestID       string
	Revision        int
	Repository      string
	Path            string
	ArgoApp         string
	BackstageTaskID string
}

// FieldError reports one missing or invalid key.
type FieldError struct {
	Key    string
	Reason string
}

func (e FieldError) Error() string { return fmt.Sprintf("%s: %s", e.Key, e.Reason) }

// Parse extracts Metadata and returns every contract violation found.
// A non-empty error list means the Claim must surface as UNKNOWN with
// INFRA.CLAIM.INVALID_METADATA (spec §17).
func Parse(labels, annotations map[string]string) (Metadata, []FieldError) {
	var errs []FieldError
	req := func(m map[string]string, k string) string {
		v := m[k]
		if v == "" {
			errs = append(errs, FieldError{k, "required"})
		}
		return v
	}
	if labels[LabelManaged] != "true" {
		errs = append(errs, FieldError{LabelManaged, `must be "true"`})
	}
	md := Metadata{
		Owner:           req(labels, LabelOwner),
		Environment:     req(labels, LabelEnvironment),
		Provider:        req(labels, LabelProvider),
		ResourceType:    req(labels, LabelResourceType),
		RequestID:       req(annotations, AnnotationRequestID),
		Repository:      req(annotations, AnnotationRepository),
		Path:            req(annotations, AnnotationPath),
		ArgoApp:         annotations[AnnotationArgoApp],
		BackstageTaskID: annotations[AnnotationBackstageTaskID],
	}
	if md.Provider != "" && !Providers[md.Provider] {
		errs = append(errs, FieldError{LabelProvider, fmt.Sprintf("unsupported provider %q", md.Provider)})
	}
	if md.RequestID != "" && !correlation.Valid(md.RequestID) {
		errs = append(errs, FieldError{AnnotationRequestID, "invalid correlation id format"})
	}
	if v := annotations[AnnotationRevision]; v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			errs = append(errs, FieldError{AnnotationRevision, "must be a non-negative integer"})
		}
		md.Revision = n
	}
	return md, errs
}
