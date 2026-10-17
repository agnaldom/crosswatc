package kube

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func fastLease(id string) LeaderConfig {
	return LeaderConfig{Namespace: "crosswatc", Identity: id, LeaseDuration: 600 * time.Millisecond,
		RenewDeadline: 400 * time.Millisecond, RetryPeriod: 100 * time.Millisecond}
}

func TestLeaderElectionSingleLeaderAndFailover(t *testing.T) {
	cs := k8sfake.NewClientset()
	coord := cs.CoordinationV1()
	var leading atomic.Int32
	started := make(chan string, 2)

	run := func(ctx context.Context, id string) chan error {
		errCh := make(chan error, 1)
		go func() {
			errCh <- RunLeaderElection(ctx, coord, fastLease(id), func(ctx context.Context) {
				if leading.Add(1) != 1 {
					t.Errorf("two leaders at once")
				}
				started <- id
				<-ctx.Done()
				leading.Add(-1)
			})
		}()
		return errCh
	}

	ctxA, cancelA := context.WithCancel(context.Background())
	errA := run(ctxA, "worker-a")
	if first := waitLeader(t, started); first != "worker-a" {
		t.Fatalf("leader %s", first)
	}
	ctxB, cancelB := context.WithCancel(context.Background())
	defer cancelB()
	errB := run(ctxB, "worker-b")

	lease, err := coord.Leases("crosswatc").Get(context.Background(), LeaseName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity != "worker-a" {
		t.Fatalf("lease holder %v", lease.Spec.HolderIdentity)
	}

	cancelA() // graceful shutdown releases the Lease
	if err := <-errA; err != nil {
		t.Fatalf("cancelled leader must return nil, got %v", err)
	}
	if got := waitLeader(t, started); got != "worker-b" {
		t.Fatalf("failover leader %s", got)
	}
	cancelB()
	if err := <-errB; err != nil {
		t.Fatal(err)
	}
}

func TestLeaderElectionLostReturnsError(t *testing.T) {
	cs := k8sfake.NewClientset()
	var outage atomic.Bool
	cs.PrependReactor("update", "leases", func(k8stesting.Action) (bool, runtime.Object, error) {
		if outage.Load() {
			return true, nil, errors.New("apiserver unavailable")
		}
		return false, nil, nil
	})
	coord := cs.CoordinationV1()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	started := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunLeaderElection(ctx, coord, fastLease("worker-a"), func(ctx context.Context) {
			started <- "worker-a"
			<-ctx.Done()
		})
	}()
	waitLeader(t, started)
	outage.Store(true) // the API server stops accepting renewals: after RenewDeadline leadership is lost
	select {
	case err := <-errCh:
		if !errors.Is(err, ErrLeadershipLost) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("leadership loss not detected")
	}
}

func waitLeader(t *testing.T, ch chan string) string {
	t.Helper()
	select {
	case id := <-ch:
		return id
	case <-time.After(5 * time.Second):
		t.Fatal("no leader elected")
		return ""
	}
}

func TestPodNamespace(t *testing.T) {
	if PodNamespace("explicit") != "explicit" {
		t.Fatal("explicit namespace must win")
	}
	t.Setenv("POD_NAMESPACE", "from-env")
	if PodNamespace("") != "from-env" {
		t.Fatal("POD_NAMESPACE ignored")
	}
}
