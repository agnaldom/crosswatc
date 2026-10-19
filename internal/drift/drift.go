// Package drift detects divergence between desired (Git), applied (Claim) and observed (XR/MR) state (spec §19, §47).
// It is pure: callers supply the signals and the previous state.
package drift

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/redact"
)

// Signal names (spec §47).
const (
	SignalArgoOutOfSync  = "argo-out-of-sync"
	SignalGitClaim       = "git-claim-mismatch"
	SignalXRNotSynced    = "xr-not-synced"
	SignalMRNotReady     = "mr-not-ready"
	SignalCloudChangeErr = "cloud-reconcile-error"
)

// Difference is one field whose desired and observed values differ.
type Difference struct {
	Field    string `json:"field"`
	Desired  any    `json:"desired"`
	Observed any    `json:"observed"`
}

// State mirrors DriftState of the spec.
type State struct {
	Detected    bool         `json:"detected"`
	Signals     []string     `json:"signals,omitempty"`
	Differences []Difference `json:"differences,omitempty"`
	DetectedAt  *time.Time   `json:"detectedAt,omitempty"`
}

// Input bundles the signals for one resource.
type Input struct {
	Signals lifecycle.Signals // Claim/XR/MRs/Argo + EverOperational + OpenRevision
	// GitParameters is spec.parameters of the manifest at the base branch; nil when Git was not read.
	GitParameters map[string]any
	// ClaimParameters is spec.parameters of the live Claim.
	ClaimParameters map[string]any
	Previous        State
	Now             time.Time
}

// Detect evaluates drift. Resources that never became operational or have a change in flight
// are not in drift: divergence there is expected.
func Detect(in Input) State {
	s := in.Signals
	if !s.EverOperational || s.OpenRevision != nil {
		return State{}
	}
	var out State
	if s.Argo.Found && s.Argo.SyncStatus == "OutOfSync" && s.Argo.OperationPhase != "Running" {
		out.Signals = append(out.Signals, SignalArgoOutOfSync)
	}
	if in.GitParameters != nil {
		out.Differences = Compare(in.GitParameters, in.ClaimParameters)
		if len(out.Differences) > 0 {
			out.Signals = append(out.Signals, SignalGitClaim)
		}
	}
	if s.XR.Present && !s.XR.Conditions.IsTrue("Synced") {
		out.Signals = append(out.Signals, SignalXRNotSynced)
	}
	for _, m := range s.MRs {
		if !m.Present || !m.Conditions.IsTrue("Ready") {
			out.Signals = append(out.Signals, SignalMRNotReady)
			break
		}
	}
	for _, m := range s.MRs {
		if c, ok := m.Conditions.Get("Synced"); ok && c.Status == "False" {
			out.Signals = append(out.Signals, SignalCloudChangeErr)
			break
		}
	}
	out.Detected = len(out.Signals) > 0
	if out.Detected {
		at := in.Now
		if in.Previous.Detected && in.Previous.DetectedAt != nil {
			at = *in.Previous.DetectedAt
		}
		out.DetectedAt = &at
	}
	return out
}

// Compare lists keys of desired (flattened) whose observed value differs. Keys only present in
// observed (API-server defaults, late-initialized fields) are not drift. Sensitive values are masked.
func Compare(desired, observed map[string]any) []Difference {
	d, o := flatten("", desired), flatten("", observed)
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []Difference
	for _, k := range keys {
		ov, ok := o[k]
		if ok && same(d[k], ov) {
			continue
		}
		diff := Difference{Field: k, Desired: d[k], Observed: ov}
		if redact.Sensitive(lastSegment(k)) {
			diff.Desired, diff.Observed = redact.Placeholder, redact.Placeholder
		}
		out = append(out, diff)
	}
	return out
}

func flatten(prefix string, m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if mm, ok := v.(map[string]any); ok && len(mm) > 0 {
			for fk, fv := range flatten(key, mm) {
				out[fk] = fv
			}
			continue
		}
		out[key] = v
	}
	return out
}

// same compares across YAML/JSON numeric representations.
func same(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func lastSegment(k string) string {
	for i := len(k) - 1; i >= 0; i-- {
		if k[i] == '.' {
			return k[i+1:]
		}
	}
	return k
}

// Transition classifies a change between previous and current detection.
func Transition(prev, cur State) (detected, resolved bool) {
	return !prev.Detected && cur.Detected, prev.Detected && !cur.Detected
}
