// Command api serves the Crosswatc REST API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/config"
	"github.com/agnaldom/crosswatc/internal/httpapi"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/logging"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(os.Stdout, cfg.LogFormat, cfg.LogLevel).With("service", "crosswatc-api")
	slog.SetDefault(log)
	log.Info("starting", "version", version.Version, "commit", version.Commit, "auth", cfg.Auth.Mode)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Migrations are owned by the worker; the API only reads.
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	mapping := auth.GroupMapping{SquadPrefix: cfg.Auth.SquadGroupPrefix, SREGroup: cfg.Auth.SREGroup}
	var authn auth.Authenticator
	if cfg.Auth.Mode == config.AuthModeDev {
		log.Warn("DEV authentication enabled: identities come from X-Dev-* headers")
		authn = auth.Dev{Mapping: mapping}
	} else {
		if authn, err = auth.NewOIDC(ctx, cfg.Auth.OIDCIssuer, cfg.Auth.OIDCAudience, mapping); err != nil {
			return err
		}
	}

	deps := httpapi.Deps{Store: st, Auth: authn, Log: log}
	if restCfg, err := kube.RESTConfig(cfg.Kubeconfig); err != nil {
		log.Warn("kubernetes not configured; catalog endpoints disabled", "err", err)
	} else if kc, err := kube.New(restCfg, cfg.K8sPageLimit); err != nil {
		return err
	} else {
		deps.Catalog = kc
	}

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.New(deps), ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Info("listening", "addr", cfg.HTTPAddr)

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shCtx)
}
