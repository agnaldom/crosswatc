package httpapi

import (
	"encoding/json"
	"time"

	"github.com/agnaldom/crosswatc/internal/store"
)

type claimRefDTO struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

type compositionDTO struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
}

type resourceDTO struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Kind            string          `json:"kind"`
	Provider        string          `json:"provider"`
	Environment     string          `json:"environment"`
	Owner           string          `json:"owner"`
	RequestedBy     string          `json:"requestedBy,omitempty"`
	Region          string          `json:"region,omitempty"`
	ResourceGroup   string          `json:"resourceGroup,omitempty"`
	ClaimRef        claimRefDTO     `json:"claimRef"`
	Repository      string          `json:"repository"`
	GitPath         string          `json:"gitPath"`
	ResourceState   string          `json:"resourceState"`
	CurrentRevision *int            `json:"currentRevision,omitempty"`
	Composition     *compositionDTO `json:"composition,omitempty"`
	CorrelationID   string          `json:"correlationId"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

func toResourceDTO(r store.TrackedResource) resourceDTO {
	d := resourceDTO{
		ID: r.ID.String(), Name: r.Name, Kind: r.Kind, Provider: r.Provider, Environment: r.Environment, Owner: r.Owner,
		RequestedBy: r.RequestedBy, Region: r.Region, ResourceGroup: r.ResourceGroup,
		ClaimRef:   claimRefDTO{Namespace: r.ClaimNamespace, Name: r.ClaimName, Kind: r.ClaimKind, APIVersion: r.ClaimAPIVersion},
		Repository: r.Repository, GitPath: r.GitPath, ResourceState: r.ResourceState, CurrentRevision: r.CurrentRevision,
		CorrelationID: r.CorrelationID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.CompositionName != "" {
		d.Composition = &compositionDTO{Name: r.CompositionName, Revision: r.CompositionRevision}
	}
	return d
}

type revisionDTO struct {
	ID            string          `json:"id"`
	ResourceID    string          `json:"resourceId"`
	Rev           int             `json:"rev"`
	Kind          string          `json:"kind"`
	State         string          `json:"state"`
	Changes       json.RawMessage `json:"changes,omitempty"`
	Branch        string          `json:"branch,omitempty"`
	CommitSHA     string          `json:"commitSha,omitempty"`
	PRNumber      *int            `json:"prNumber,omitempty"`
	PRURL         string          `json:"prUrl,omitempty"`
	ArgoApp       string          `json:"argoApp,omitempty"`
	FailureReason json.RawMessage `json:"failureReason,omitempty"`
	StartedAt     *time.Time      `json:"startedAt,omitempty"`
	CompletedAt   *time.Time      `json:"completedAt,omitempty"`
}

func toRevisionDTO(r store.Revision) revisionDTO {
	return revisionDTO{
		ID: r.ID.String(), ResourceID: r.ResourceID.String(), Rev: r.Rev, Kind: r.Kind, State: r.State, Changes: r.Changes,
		Branch: r.Branch, CommitSHA: r.CommitSHA, PRNumber: r.PRNumber, PRURL: r.PRURL, ArgoApp: r.ArgoApp,
		FailureReason: r.FailureReason, StartedAt: r.StartedAt, CompletedAt: r.CompletedAt,
	}
}

type operationDTO struct {
	ID            string     `json:"id"`
	ResourceID    string     `json:"resourceId"`
	RevisionID    string     `json:"revisionId,omitempty"`
	CorrelationID string     `json:"correlationId"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	RequestedBy   string     `json:"requestedBy"`
	RequestedAt   time.Time  `json:"requestedAt"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
}

func toOperationDTO(o store.Operation) operationDTO {
	d := operationDTO{
		ID: o.ID.String(), ResourceID: o.ResourceID.String(), CorrelationID: o.CorrelationID, Type: o.Type, Status: o.Status,
		RequestedBy: o.RequestedBy, RequestedAt: o.RequestedAt, StartedAt: o.StartedAt, FinishedAt: o.FinishedAt,
	}
	if o.RevisionID != nil {
		d.RevisionID = o.RevisionID.String()
	}
	return d
}

type eventDTO struct {
	EventID       string    `json:"eventId"`
	CorrelationID string    `json:"correlationId,omitempty"`
	ResourceID    string    `json:"resourceId,omitempty"`
	RevisionID    string    `json:"revisionId,omitempty"`
	OperationID   string    `json:"operationId,omitempty"`
	Source        string    `json:"source"`
	Type          string    `json:"type"`
	Level         string    `json:"level"`
	Message       string    `json:"message,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

func toEventDTO(e store.Event) eventDTO {
	d := eventDTO{EventID: e.ID.String(), CorrelationID: e.CorrelationID, Source: e.Source, Type: e.Type, Level: e.Level,
		Message: e.Message, Timestamp: e.CreatedAt}
	if e.ResourceID != nil {
		d.ResourceID = e.ResourceID.String()
	}
	if e.RevisionID != nil {
		d.RevisionID = e.RevisionID.String()
	}
	if e.OperationID != nil {
		d.OperationID = e.OperationID.String()
	}
	return d
}

type page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

func newPage[T any](items []T, next string) page[T] {
	if items == nil {
		items = []T{}
	}
	p := page[T]{Items: items}
	if next != "" {
		p.NextCursor = &next
	}
	return p
}
