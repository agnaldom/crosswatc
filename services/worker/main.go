// Command worker runs discovery, informers, adapters and the state reconciler on the elected leader (spec §35.2).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/agnaldom/crosswatc/internal/adapters/argocd"
	"github.com/agnaldom/crosswatc/internal/adapters/backstage"
	"github.com/agnaldom/crosswatc/internal/adapters/github"
	"github.com/agnaldom/crosswatc/internal/config"
	"github.com/agnaldom/crosswatc/internal/informer"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/logging"
	"github.com/agnaldom/crosswatc/internal/metrics"
	"github.com/agnaldom/crosswatc/internal/reconciler"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/tracing"
	"github.com/agnaldom/crosswatc/internal/version"
	"github.com/agnaldom/crosswatc/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(os.Stdout, cfg.LogFormat, cfg.LogLevel).With("service", "crosswatc-worker")
	slog.SetDefault(log)
	log.Info("starting", "version", version.Version, "commit", version.Commit)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := tracing.Setup(ctx, "crosswatc-worker")
	if err != nil {
		return err
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	if err := store.Migrate(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	restCfg, err := kube.RESTConfig(cfg.Kubeconfig)
	if err != nil {
		return err
	}
	kc, err := kube.New(restCfg, cfg.K8sPageLimit)
	if err != nil {
		return err
	}

	rec := &reconciler.Reconciler{
		Discovery:    kc,
		Argo:         argocd.New(kc, cfg.ArgoCDNamespace, kube.IsNotFound),
		Store:        st,
		Log:          log,
		StuckTimeout: cfg.StuckTimeout,
	}
	if cfg.GitHub.AppID != 0 {
		gh, err := github.New(github.Config{APIURL: cfg.GitHub.APIURL, AppID: cfg.GitHub.AppID,
			InstallationID: cfg.GitHub.InstallationID, PrivateKeyPath: cfg.GitHub.PrivateKeyPath})
		if err != nil {
			return err
		}
		rec.PullRequests = gh
		rec.Git, rec.BaseBranch, rec.GitCacheTTL = gh, cfg.GitOpsBaseBranch, cfg.ResyncInterval
	} else {
		log.Warn("GitHub App not configured; PR and CI stages will be skipped")
	}
	if cfg.Backstage.URL != "" {
		rec.Backstage = backstage.New(cfg.Backstage.URL, cfg.Backstage.Token, cfg.Backstage.PollInterval, nil)
	}

	runner := &worker.Runner{
		Rec: rec, Resync: cfg.ResyncInterval, Log: log, Inbox: st, InboxInterval: cfg.InboxPollInterval, Stream: st,
		Watch: informer.Options{Secrets: cfg.Informers.Secrets, Events: cfg.Informers.Events},
	}
	if cfg.Informers.Enabled {
		runner.Informers = informer.New(kc.Dynamic, kc.Metadata, log, runner.OnEvent)
	}

	// Standby replicas are ready once the DB answers; the leader also needs a resync and synced informers.
	var leading atomic.Bool
	ready := func() bool { return !leading.Load() || runner.Ready() }
	srv := probeServer(":8081", st, ready)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("probe server", "err", err)
		}
	}()
	defer func() {
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()

	lead := func(ctx context.Context) {
		leading.Store(true)
		defer leading.Store(false)
		log.Info("leading: starting informers, reconciler and polling")
		if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("runner stopped", "err", err)
		}
	}
	if !cfg.LeaderElection.Enabled {
		log.Warn("leader election disabled: run a single worker replica")
		lead(ctx)
		log.Info("shutting down")
		return nil
	}

	coord, err := kube.CoordinationClient(restCfg)
	if err != nil {
		return err
	}
	identity := cfg.LeaderElection.Identity
	if identity == "" {
		if identity, err = os.Hostname(); err != nil {
			return err
		}
	}
	le := cfg.LeaderElection
	err = kube.RunLeaderElection(ctx, coord, kube.LeaderConfig{
		Namespace: kube.PodNamespace(le.Namespace), Identity: identity,
		LeaseDuration: le.LeaseDuration, RenewDeadline: le.RenewDeadline, RetryPeriod: le.RetryPeriod,
		OnLeaderChange: func(id string) { log.Info("leader elected", "leader", id, "self", id == identity) },
	}, lead)
	log.Info("shutting down")
	return err
}

func probeServer(addr string, st *store.Store, ready func() bool) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("GET /metrics", metrics.Handler())
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready() || st.Ping(r.Context()) != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}
