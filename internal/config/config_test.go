package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	c, err := LoadFrom(env(map[string]string{"CROSSWATC_AUTH_MODE": "dev"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Env != EnvDevelopment || c.HTTPAddr != ":8080" || c.K8sPageLimit != 500 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if c.ResyncInterval != 5*time.Minute || c.StuckTimeout != 30*time.Minute || c.Backstage.PollInterval != 30*time.Second {
		t.Fatalf("unexpected durations: %+v", c)
	}
}

func TestLeaderElectionAndInformerDefaults(t *testing.T) {
	c, err := LoadFrom(env(map[string]string{"CROSSWATC_AUTH_MODE": "dev"}))
	if err != nil {
		t.Fatal(err)
	}
	le := c.LeaderElection
	if !le.Enabled || le.LeaseDuration != 15*time.Second || le.RenewDeadline != 10*time.Second || le.RetryPeriod != 2*time.Second {
		t.Fatalf("leader election: %+v", le)
	}
	if c.InboxPollInterval != time.Second || c.ArgoCDWebhookToken != "" {
		t.Fatalf("inbox/webhook defaults: %+v", c)
	}
	if !c.Informers.Enabled || !c.Informers.Secrets || !c.Informers.Events {
		t.Fatalf("informers: %+v", c.Informers)
	}
	c, err = LoadFrom(env(map[string]string{"CROSSWATC_AUTH_MODE": "dev", "CROSSWATC_LEADER_ELECTION": "false",
		"CROSSWATC_LEADER_ELECTION_RENEW_DEADLINE": "1h", "CROSSWATC_WATCH_SECRETS": "false"}))
	if err != nil {
		t.Fatal(err) // ordering is only validated when enabled
	}
	if c.LeaderElection.Enabled || c.Informers.Secrets {
		t.Fatalf("overrides ignored: %+v", c)
	}
}

func TestDevAuthForbiddenInProduction(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"CROSSWATC_ENV": "production", "CROSSWATC_AUTH_MODE": "dev"}))
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestOIDCRequiredInProduction(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"CROSSWATC_ENV": "production"}))
	if err == nil {
		t.Fatal("expected error when OIDC issuer/audience missing in production")
	}
	_, err = LoadFrom(env(map[string]string{
		"CROSSWATC_ENV":           "production",
		"CROSSWATC_OIDC_ISSUER":   "https://idp.example.com",
		"CROSSWATC_OIDC_AUDIENCE": "crosswatc",
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidValues(t *testing.T) {
	for name, m := range map[string]map[string]string{
		"env":      {"CROSSWATC_ENV": "staging"},
		"duration": {"CROSSWATC_RESYNC_INTERVAL": "soon"},
		"int":      {"CROSSWATC_K8S_PAGE_LIMIT": "x"},
		"limit":    {"CROSSWATC_K8S_PAGE_LIMIT": "0"},
		"auth":     {"CROSSWATC_AUTH_MODE": "basic"},
		"bool":     {"CROSSWATC_LEADER_ELECTION": "maybe"},
		"lease":    {"CROSSWATC_LEADER_ELECTION_RENEW_DEADLINE": "20s"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadFrom(env(m)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBackstageURLTrimmed(t *testing.T) {
	c, err := LoadFrom(env(map[string]string{"CROSSWATC_AUTH_MODE": "dev", "CROSSWATC_BACKSTAGE_URL": "https://bs.example.com/"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Backstage.URL != "https://bs.example.com" {
		t.Fatalf("got %q", c.Backstage.URL)
	}
}
