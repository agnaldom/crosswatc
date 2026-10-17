// Command worker runs discovery, adapters and the state reconciler.
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
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/logging"
	"github.com/agnaldom/crosswatc/internal/reconciler"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/version"
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
	} else {
		log.Warn("GitHub App not configured; PR and CI stages will be skipped")
	}
	if cfg.Backstage.URL != "" {
		rec.Backstage = backstage.New(cfg.Backstage.URL, cfg.Backstage.Token, cfg.Backstage.PollInterval, nil)
	}

	var ready atomic.Bool
	srv := probeServer(":8081", st, &ready)
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

	// Single replica until leader election lands (P2.1, §35.2).
	ticker := time.NewTicker(cfg.ResyncInterval)
	defer ticker.Stop()
	for {
		start := time.Now()
		res, err := rec.RunOnce(ctx)
		if err != nil {
			log.Error("resync finished with errors", "err", err, "result", res, "took", time.Since(start))
		} else {
			ready.Store(true)
			log.Info("resync finished", "result", res, "took", time.Since(start))
		}
		select {
		case <-ctx.Done():
			log.Info("shutting down")
			return nil
		case <-ticker.C:
		}
	}
}

func probeServer(addr string, st *store.Store, ready *atomic.Bool) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() || st.Ping(r.Context()) != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}
