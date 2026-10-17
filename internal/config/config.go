// Package config loads Crosswatc runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Env string

const (
	EnvDevelopment Env = "development"
	EnvProduction  Env = "production"
)

type AuthMode string

const (
	AuthModeOIDC AuthMode = "oidc"
	AuthModeDev  AuthMode = "dev"
)

type Config struct {
	Env       Env
	HTTPAddr  string
	LogLevel  string
	LogFormat string

	DatabaseURL string
	Kubeconfig  string // empty → in-cluster

	Auth      AuthConfig
	GitHub    GitHubConfig
	Backstage BackstageConfig

	ArgoCDNamespace string

	ResyncInterval time.Duration
	StuckTimeout   time.Duration
	K8sPageLimit   int64
}

type AuthConfig struct {
	Mode             AuthMode
	OIDCIssuer       string
	OIDCAudience     string
	SquadGroupPrefix string
	SREGroup         string
}

type GitHubConfig struct {
	APIURL         string
	AppID          int64
	InstallationID int64
	PrivateKeyPath string
	WebhookSecret  string
}

type BackstageConfig struct {
	URL          string
	Token        string
	PollInterval time.Duration
}

// Load reads configuration using os.Getenv.
func Load() (Config, error) { return LoadFrom(os.Getenv) }

// LoadFrom reads configuration using the given lookup function (testable).
func LoadFrom(getenv func(string) string) (Config, error) {
	r := reader{getenv: getenv}
	c := Config{
		Env:         Env(r.str("CROSSWATC_ENV", string(EnvDevelopment))),
		HTTPAddr:    r.str("CROSSWATC_HTTP_ADDR", ":8080"),
		LogLevel:    r.str("CROSSWATC_LOG_LEVEL", "info"),
		LogFormat:   r.str("CROSSWATC_LOG_FORMAT", "json"),
		DatabaseURL: r.str("CROSSWATC_DATABASE_URL", ""),
		Kubeconfig:  r.str("KUBECONFIG", ""),
		Auth: AuthConfig{
			Mode:             AuthMode(r.str("CROSSWATC_AUTH_MODE", string(AuthModeOIDC))),
			OIDCIssuer:       r.str("CROSSWATC_OIDC_ISSUER", ""),
			OIDCAudience:     r.str("CROSSWATC_OIDC_AUDIENCE", ""),
			SquadGroupPrefix: r.str("CROSSWATC_SQUAD_GROUP_PREFIX", "squad-"),
			SREGroup:         r.str("CROSSWATC_SRE_GROUP", "platform-sre"),
		},
		GitHub: GitHubConfig{
			APIURL:         r.str("CROSSWATC_GITHUB_API_URL", "https://api.github.com"),
			AppID:          r.int64("CROSSWATC_GITHUB_APP_ID", 0),
			InstallationID: r.int64("CROSSWATC_GITHUB_INSTALLATION_ID", 0),
			PrivateKeyPath: r.str("CROSSWATC_GITHUB_PRIVATE_KEY_PATH", ""),
			WebhookSecret:  r.str("CROSSWATC_GITHUB_WEBHOOK_SECRET", ""),
		},
		Backstage: BackstageConfig{
			URL:          strings.TrimRight(r.str("CROSSWATC_BACKSTAGE_URL", ""), "/"),
			Token:        r.str("CROSSWATC_BACKSTAGE_TOKEN", ""),
			PollInterval: r.duration("CROSSWATC_BACKSTAGE_POLL_INTERVAL", 30*time.Second),
		},
		ArgoCDNamespace: r.str("CROSSWATC_ARGOCD_NAMESPACE", "argocd"),
		ResyncInterval:  r.duration("CROSSWATC_RESYNC_INTERVAL", 5*time.Minute),
		StuckTimeout:    r.duration("CROSSWATC_STUCK_TIMEOUT", 30*time.Minute),
		K8sPageLimit:    r.int64("CROSSWATC_K8S_PAGE_LIMIT", 500),
	}
	if len(r.errs) > 0 {
		return Config{}, errors.Join(r.errs...)
	}
	return c, c.Validate()
}

// Validate enforces cross-field rules from the spec.
func (c Config) Validate() error {
	var errs []error
	switch c.Env {
	case EnvDevelopment, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("CROSSWATC_ENV: invalid value %q", c.Env))
	}
	switch c.Auth.Mode {
	case AuthModeDev:
		if c.Env == EnvProduction {
			errs = append(errs, errors.New("CROSSWATC_AUTH_MODE=dev is forbidden when CROSSWATC_ENV=production"))
		}
	case AuthModeOIDC:
		if c.Env == EnvProduction && (c.Auth.OIDCIssuer == "" || c.Auth.OIDCAudience == "") {
			errs = append(errs, errors.New("CROSSWATC_OIDC_ISSUER and CROSSWATC_OIDC_AUDIENCE are required in production"))
		}
	default:
		errs = append(errs, fmt.Errorf("CROSSWATC_AUTH_MODE: invalid value %q", c.Auth.Mode))
	}
	if c.K8sPageLimit <= 0 {
		errs = append(errs, errors.New("CROSSWATC_K8S_PAGE_LIMIT must be > 0"))
	}
	return errors.Join(errs...)
}

type reader struct {
	getenv func(string) string
	errs   []error
}

func (r *reader) str(key, def string) string {
	if v := strings.TrimSpace(r.getenv(key)); v != "" {
		return v
	}
	return def
}

func (r *reader) int64(key string, def int64) int64 {
	v := r.getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: %w", key, err))
	}
	return n
}

func (r *reader) duration(key string, def time.Duration) time.Duration {
	v := r.getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: %w", key, err))
	}
	return d
}
