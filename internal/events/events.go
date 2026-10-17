// Package events is the Canonical Event Model (spec §33, §34) and the normalizers that turn GitHub and
// Argo CD webhook payloads into canonical events (§35). Webhooks are triggers, not truth (§35.1): the
// worker uses the normalized hints to pick the resources to re-read from the source.
package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/agnaldom/crosswatc/internal/correlation"
)

// Type is a canonical event type (§33).
type Type string

const (
	RequestCreated             Type = "REQUEST.CREATED"
	RequestScaffolderCompleted Type = "REQUEST.SCAFFOLDER.COMPLETED"
	RequestScaffolderFailed    Type = "REQUEST.SCAFFOLDER.FAILED"

	GitOpsPRCreated      Type = "GITOPS.PR.CREATED"
	GitOpsPRClosed       Type = "GITOPS.PR.CLOSED"
	GitOpsCheckSucceeded Type = "GITOPS.CHECK.SUCCEEDED"
	GitOpsPRApproved     Type = "GITOPS.PR.APPROVED"
	GitOpsPRMerged       Type = "GITOPS.PR.MERGED"
	GitOpsCheckFailed    Type = "GITOPS.CHECK.FAILED"

	GitOpsSyncStarted   Type = "GITOPS.SYNC.STARTED"
	GitOpsSyncSucceeded Type = "GITOPS.SYNC.SUCCEEDED"
	GitOpsSyncFailed    Type = "GITOPS.SYNC.FAILED"

	InfraClaimCreated         Type = "INFRA.CLAIM.CREATED"
	InfraClaimDeleted         Type = "INFRA.CLAIM.DELETED"
	InfraClaimInvalidMetadata Type = "INFRA.CLAIM.INVALID_METADATA"
	InfraXRCreated            Type = "INFRA.XR.CREATED"

	InfraReconcileStarted Type = "INFRA.RECONCILE.STARTED"
	InfraReconcileReady   Type = "INFRA.RECONCILE.READY"
	InfraReconcileFailed  Type = "INFRA.RECONCILE.FAILED"

	InfraManagedResourceCreated Type = "INFRA.MANAGED_RESOURCE.CREATED"
	InfraManagedResourceReady   Type = "INFRA.MANAGED_RESOURCE.READY"
	InfraManagedResourceFailed  Type = "INFRA.MANAGED_RESOURCE.FAILED"

	LifecycleOperational  Type = "LIFECYCLE.OPERATIONAL"
	LifecycleDegraded     Type = "LIFECYCLE.DEGRADED"
	LifecycleDriftDetect  Type = "LIFECYCLE.DRIFT.DETECTED"
	LifecycleDriftResolve Type = "LIFECYCLE.DRIFT.RESOLVED"

	OperationStuck Type = "OPERATION.STUCK"
)

// Level of an event (§33).
type Level string

const (
	Info    Level = "info"
	Warning Level = "warning"
	Error   Level = "error"
)

// Status of an event (§33).
type Status string

const (
	Success    Status = "success"
	Failure    Status = "failure"
	InProgress Status = "in_progress"
)

// Sources.
const (
	SourceGitHub     = "github"
	SourceArgoCD     = "argocd"
	SourceCrossplane = "crossplane"
	SourceKubernetes = "kubernetes"
	SourceBackstage  = "backstage"
	SourceCrosswatc  = "crosswatc"
)

type meta struct {
	level  Level
	status Status
}

var catalog = map[Type]meta{
	RequestCreated: {Info, InProgress}, RequestScaffolderCompleted: {Info, Success}, RequestScaffolderFailed: {Error, Failure},
	GitOpsPRCreated: {Info, InProgress}, GitOpsPRClosed: {Warning, Failure}, GitOpsCheckSucceeded: {Info, Success},
	GitOpsPRApproved: {Info, Success}, GitOpsPRMerged: {Info, Success}, GitOpsCheckFailed: {Error, Failure},
	GitOpsSyncStarted: {Info, InProgress}, GitOpsSyncSucceeded: {Info, Success}, GitOpsSyncFailed: {Error, Failure},
	InfraClaimCreated: {Info, Success}, InfraClaimDeleted: {Warning, Success}, InfraClaimInvalidMetadata: {Warning, Failure},
	InfraXRCreated:        {Info, Success},
	InfraReconcileStarted: {Info, InProgress}, InfraReconcileReady: {Info, Success}, InfraReconcileFailed: {Error, Failure},
	InfraManagedResourceCreated: {Info, InProgress}, InfraManagedResourceReady: {Info, Success}, InfraManagedResourceFailed: {Error, Failure},
	LifecycleOperational: {Info, Success}, LifecycleDegraded: {Warning, Failure},
	LifecycleDriftDetect: {Warning, Failure}, LifecycleDriftResolve: {Info, Success},
	OperationStuck: {Warning, Failure},
}

// Known reports whether t is a canonical type.
func Known(t Type) bool { _, ok := catalog[t]; return ok }

// DefaultLevel is the level of t (info for unknown types).
func DefaultLevel(t Type) Level {
	if m, ok := catalog[t]; ok {
		return m.level
	}
	return Info
}

// DefaultStatus is the status of t.
func DefaultStatus(t Type) Status {
	if m, ok := catalog[t]; ok {
		return m.status
	}
	return InProgress
}

// Event is the canonical event (§34).
type Event struct {
	EventID       string            `json:"eventId"`
	CorrelationID string            `json:"correlationId,omitempty"`
	ResourceID    string            `json:"resourceId,omitempty"`
	RevisionID    string            `json:"revisionId,omitempty"`
	OperationID   string            `json:"operationId,omitempty"`
	Source        string            `json:"source"`
	Type          Type              `json:"type"`
	Level         Level             `json:"level"`
	Status        Status            `json:"status"`
	Timestamp     time.Time         `json:"timestamp"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// New builds an event with the catalog level/status of t.
func New(id, source string, t Type, ts time.Time) Event {
	return Event{EventID: id, Source: source, Type: t, Level: DefaultLevel(t), Status: DefaultStatus(t), Timestamp: ts.UTC()}
}

// Validate checks required fields and the closed vocabularies of §33.
func (e Event) Validate() error {
	switch {
	case e.EventID == "":
		return fmt.Errorf("events: eventId required")
	case e.Source == "":
		return fmt.Errorf("events: source required")
	case !Known(e.Type):
		return fmt.Errorf("events: unknown type %q", e.Type)
	}
	switch e.Level {
	case Info, Warning, Error:
	default:
		return fmt.Errorf("events: invalid level %q", e.Level)
	}
	switch e.Status {
	case Success, Failure, InProgress:
	default:
		return fmt.Errorf("events: invalid status %q", e.Status)
	}
	return nil
}

// Hints locate the resources a webhook refers to.
type Hints struct {
	Repository     string   // owner/name
	CorrelationIDs []string // §16 IDs found in PR title/body/branch or commit message
	PRNumber       int
	ArgoApp        string // namespace/name
}

// Normalized is the result of normalizing one webhook delivery.
type Normalized struct {
	Events []Event
	Hints  Hints
}

// GitHubEvents are the GitHub event names accepted by the webhook endpoint (§35).
var GitHubEvents = map[string]bool{"pull_request": true, "check_run": true, "pull_request_review": true, "push": true}

type ghPayload struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest *struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		Merged  bool   `json:"merged"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
	} `json:"pull_request"`
	Review *struct {
		State string `json:"state"`
	} `json:"review"`
	CheckRun *struct {
		Name         string `json:"name"`
		Status       string `json:"status"`
		Conclusion   string `json:"conclusion"`
		HeadSHA      string `json:"head_sha"`
		PullRequests []struct {
			Number int `json:"number"`
			Head   struct {
				Ref string `json:"ref"`
			} `json:"head"`
		} `json:"pull_requests"`
	} `json:"check_run"`
	Ref        string `json:"ref"`
	HeadCommit *struct {
		Message string `json:"message"`
	} `json:"head_commit"`
}

// FromGitHub normalizes a GitHub delivery (X-GitHub-Event, X-GitHub-Delivery, body).
func FromGitHub(event, deliveryID string, body []byte, received time.Time) (Normalized, error) {
	if !GitHubEvents[event] {
		return Normalized{}, fmt.Errorf("events: unsupported github event %q", event)
	}
	var p ghPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return Normalized{}, fmt.Errorf("events: github payload: %w", err)
	}
	n := Normalized{Hints: Hints{Repository: p.Repository.FullName}}
	add := func(t Type, md map[string]string) {
		e := New(SourceGitHub+":"+deliveryID+":"+string(t), SourceGitHub, t, received)
		e.Metadata = md
		n.Events = append(n.Events, e)
	}
	var texts []string
	switch event {
	case "pull_request", "pull_request_review":
		if p.PullRequest == nil {
			return n, nil
		}
		pr := p.PullRequest
		n.Hints.PRNumber = pr.Number
		texts = append(texts, pr.Title, pr.Body, pr.Head.Ref)
		md := map[string]string{"repository": p.Repository.FullName, "pr": fmt.Sprint(pr.Number), "prUrl": pr.HTMLURL}
		if event == "pull_request_review" {
			if p.Action == "submitted" && p.Review != nil && strings.EqualFold(p.Review.State, "approved") {
				add(GitOpsPRApproved, md)
			}
			break
		}
		switch {
		case p.Action == "opened" || p.Action == "reopened":
			add(GitOpsPRCreated, md)
		case p.Action == "closed" && pr.Merged:
			add(GitOpsPRMerged, md)
		case p.Action == "closed":
			add(GitOpsPRClosed, md)
		}
	case "check_run":
		if p.CheckRun == nil {
			return n, nil
		}
		cr := p.CheckRun
		for _, pr := range cr.PullRequests {
			if n.Hints.PRNumber == 0 {
				n.Hints.PRNumber = pr.Number
			}
			texts = append(texts, pr.Head.Ref)
		}
		md := map[string]string{"repository": p.Repository.FullName, "check": cr.Name, "sha": cr.HeadSHA}
		if p.Action == "completed" || cr.Status == "completed" {
			switch cr.Conclusion {
			case "success", "neutral", "skipped":
				add(GitOpsCheckSucceeded, md)
			case "failure", "timed_out", "cancelled", "action_required", "startup_failure":
				add(GitOpsCheckFailed, md)
			}
		}
	case "push":
		texts = append(texts, p.Ref)
		if p.HeadCommit != nil {
			texts = append(texts, p.HeadCommit.Message)
		}
	}
	n.Hints.CorrelationIDs = correlation.FindAll(strings.Join(texts, "\n"))
	return n, nil
}

// ArgoPayload is the JSON body expected from argocd-notifications (webhook service template):
//
//	{"eventId":"{{.app.metadata.uid}}-{{.app.status.operationState.startedAt}}-sync-succeeded",
//	 "app":"{{.app.metadata.name}}","namespace":"{{.app.metadata.namespace}}",
//	 "event":"sync-succeeded","revision":"{{.app.status.sync.revision}}","timestamp":"{{.app.status.operationState.finishedAt}}"}
type ArgoPayload struct {
	EventID   string `json:"eventId"`
	App       string `json:"app"`
	Namespace string `json:"namespace"`
	Event     string `json:"event"`
	Revision  string `json:"revision"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

var argoTypes = map[string]Type{
	"sync-running":    GitOpsSyncStarted,
	"sync-succeeded":  GitOpsSyncSucceeded,
	"sync-failed":     GitOpsSyncFailed,
	"health-degraded": LifecycleDegraded,
}

// ParseArgo decodes and validates an Argo CD notification body.
func ParseArgo(body []byte) (ArgoPayload, error) {
	var p ArgoPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return p, fmt.Errorf("events: argocd payload: %w", err)
	}
	switch {
	case p.EventID == "":
		return p, fmt.Errorf("events: argocd payload: eventId required")
	case p.App == "":
		return p, fmt.Errorf("events: argocd payload: app required")
	case argoTypes[p.Event] == "":
		return p, fmt.Errorf("events: argocd payload: unsupported event %q", p.Event)
	}
	return p, nil
}

// FromArgo normalizes an Argo CD notification.
func FromArgo(body []byte, received time.Time) (Normalized, error) {
	p, err := ParseArgo(body)
	if err != nil {
		return Normalized{}, err
	}
	ts := received
	if t, err := time.Parse(time.RFC3339, p.Timestamp); err == nil {
		ts = t
	}
	e := New(SourceArgoCD+":"+p.EventID, SourceArgoCD, argoTypes[p.Event], ts)
	e.Metadata = map[string]string{"app": p.App, "namespace": p.Namespace, "revision": p.Revision}
	app := p.App
	if p.Namespace != "" {
		app = p.Namespace + "/" + p.App
	}
	return Normalized{Events: []Event{e}, Hints: Hints{ArgoApp: app}}, nil
}
