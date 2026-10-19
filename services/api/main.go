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

	"github.com/agnaldom/crosswatc/internal/adapters/github"
	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/config"
	"github.com/agnaldom/crosswatc/internal/cost"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/httpapi"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/logging"
	"github.com/agnaldom/crosswatc/internal/metrics"
	"github.com/agnaldom/crosswatc/internal/policy"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/stream"
	"github.com/agnaldom/crosswatc/internal/tracing"
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

	shutdownTracing, err := tracing.Setup(ctx, "crosswatc-api")
	if err != nil {
		return err
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: metrics.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server", "err", err)
		}
	}()
	defer func() { _ = metricsSrv.Close() }()

	// Migrations are owned by the worker; the API only reads.
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	mapping := auth.GroupMapping{SquadPrefix: cfg.Auth.SquadGroupPrefix, SREGroup: cfg.Auth.SREGroup, GroupsClaim: cfg.Auth.GroupsClaim}
	var authn auth.Authenticator
	if cfg.Auth.Mode == config.AuthModeDev {
		log.Warn("DEV authentication enabled: identities come from X-Dev-* headers")
		authn = auth.Dev{Mapping: mapping}
	} else {
		if authn, err = auth.NewOIDC(ctx, cfg.Auth.OIDCIssuer, cfg.Auth.OIDCAudience, mapping); err != nil {
			return err
		}
	}

	deps := httpapi.Deps{Store: st, Auth: authn, Log: log, Inbox: st, Audit: st, MutationsPerMinute: cfg.MutationsPerMinute,
		DecommissionEnvironments: cfg.DecommissionEnvironments, ProductionEnvironments: cfg.ProductionEnvironments,
		Webhooks: httpapi.WebhookConfig{GitHubSecret: cfg.GitHub.WebhookSecret, ArgoCDToken: cfg.ArgoCDWebhookToken}}
	hub := stream.NewHub(st, log, 0)
	go func() {
		if err := hub.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("stream hub stopped", "err", err)
		}
	}()
	deps.Stream = hub
	if cfg.GitHub.WebhookSecret == "" {
		log.Warn("CROSSWATC_GITHUB_WEBHOOK_SECRET not set; POST /webhooks/github disabled")
	}
	if cfg.ArgoCDWebhookToken == "" {
		log.Warn("CROSSWATC_ARGOCD_WEBHOOK_TOKEN not set; POST /webhooks/argocd disabled")
	}
	engine, err := policy.Load(ctx, cfg.PolicyDir)
	if err != nil {
		return err
	}
	log.Info("policies loaded", "modules", engine.Modules())
	deps.Policies = []changes.Policy{engine}
	if cfg.CostProvider == "azure" {
		deps.Costs = &cost.Estimator{Prices: &cost.Azure{Currency: cfg.CostCurrency}}
	}

	if cfg.GitHub.AppID != 0 {
		gh, err := github.New(github.Config{APIURL: cfg.GitHub.APIURL, AppID: cfg.GitHub.AppID,
			InstallationID: cfg.GitHub.InstallationID, PrivateKeyPath: cfg.GitHub.PrivateKeyPath})
		if err != nil {
			return err
		}
		deps.Changes, deps.GitOps = st, &gitops.Orchestrator{Repo: gh, BaseBranch: cfg.GitOpsBaseBranch}
	} else {
		log.Warn("GitHub App not configured; POST /resources/{id}/changes disabled")
	}
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
