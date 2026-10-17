package kube

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// LeaseName is the worker leader-election Lease (spec §35.2).
const LeaseName = "crosswatc-worker"

// ErrLeadershipLost is returned when this replica stops being leader while its context is still alive.
// The process should exit so the pod restarts as a clean candidate.
var ErrLeadershipLost = errors.New("kube: leader election lost")

// LeaderConfig configures RunLeaderElection.
type LeaderConfig struct {
	Namespace     string
	Name          string // defaults to LeaseName
	Identity      string
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
	// OnLeaderChange is called with the identity of the current leader (optional).
	OnLeaderChange func(identity string)
}

// CoordinationClient builds the typed client used for Leases (the only write the worker performs, §27).
func CoordinationClient(cfg *rest.Config) (coordinationv1.CoordinationV1Interface, error) {
	return coordinationv1.NewForConfig(cfg)
}

// RunLeaderElection blocks while campaigning for the Lease and calls lead with a context that is cancelled
// when leadership is lost. It returns nil when ctx is cancelled and ErrLeadershipLost otherwise.
// The Lease is released on cancellation so a standby replica takes over quickly.
func RunLeaderElection(ctx context.Context, client coordinationv1.CoordinationV1Interface, c LeaderConfig, lead func(ctx context.Context)) error {
	if c.Name == "" {
		c.Name = LeaseName
	}
	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Namespace: c.Namespace, Name: c.Name},
		Client:     client,
		LockConfig: resourcelock.ResourceLockConfig{Identity: c.Identity},
	}
	led := make(chan struct{})
	done := make(chan struct{})
	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   c.LeaseDuration,
		RenewDeadline:   c.RenewDeadline,
		RetryPeriod:     c.RetryPeriod,
		ReleaseOnCancel: true,
		Name:            c.Name,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				close(led)
				defer close(done)
				lead(ctx)
			},
			OnStoppedLeading: func() {},
			OnNewLeader: func(id string) {
				if c.OnLeaderChange != nil {
					c.OnLeaderChange(id)
				}
			},
		},
	})
	if err != nil {
		return err
	}
	le.Run(ctx)
	select {
	case <-led:
		<-done // lead observes its cancelled context; wait so callers never overlap leaders in-process
	default:
	}
	if ctx.Err() != nil {
		return nil
	}
	return ErrLeadershipLost
}

// PodNamespace returns the namespace of the running pod: explicit value, POD_NAMESPACE, the service
// account namespace file, or "default" (local development).
func PodNamespace(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	if b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(b)); ns != "" {
			return ns
		}
	}
	return "default"
}
