package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

type providerManager interface {
	ProviderConfigs(context.Context) ([]kube.ProviderConfigSummary, error)
	InstallProvider(context.Context, string, string, string) (kube.Package, error)
}

type providerCatalogEntry struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Repository    string   `json:"repository"`
	Documentation string   `json:"documentation"`
	Packages      []string `json:"packages"`
	Version       string   `json:"version"`
}

// Curated repositories only: a browser cannot install arbitrary controller images.
var providerCatalog = []providerCatalogEntry{
	{"azure", "Microsoft Azure", "Complete Azure provider bundle: shared configuration and all service controllers.", "xpkg.crossplane.io/crossplane-contrib/provider-family-azure", "https://github.com/crossplane-contrib/provider-upjet-azure", providerBundles["azure"], "v2.8.0"},
	{"aws", "Amazon Web Services", "Complete AWS provider bundle: shared configuration and all service controllers.", "xpkg.crossplane.io/crossplane-contrib/provider-family-aws", "https://github.com/crossplane-contrib/provider-upjet-aws", providerBundles["aws"], "v2.9.0"},
	{"gcp", "Google Cloud", "Complete Google Cloud provider bundle: shared configuration and all service controllers.", "xpkg.crossplane.io/crossplane-contrib/provider-family-gcp", "https://github.com/crossplane-contrib/provider-upjet-gcp", providerBundles["gcp"], "v3.1.0"},
	{"kubernetes", "Kubernetes", "Manage Kubernetes objects on target clusters.", "xpkg.crossplane.io/crossplane-contrib/provider-kubernetes", "https://github.com/crossplane-contrib/provider-kubernetes", []string{"xpkg.crossplane.io/crossplane-contrib/provider-kubernetes"}, "v1.4.1"},
	{"helm", "Helm", "Manage Helm releases on Kubernetes clusters.", "xpkg.crossplane.io/crossplane-contrib/provider-helm", "https://github.com/crossplane-contrib/provider-helm", []string{"xpkg.crossplane.io/crossplane-contrib/provider-helm"}, "v1.5.0"},
	{"sql", "SQL", "Manage SQL databases, roles and grants.", "xpkg.crossplane.io/crossplane-contrib/provider-sql", "https://github.com/crossplane-contrib/provider-sql", []string{"xpkg.crossplane.io/crossplane-contrib/provider-sql"}, "v0.16.2"},
}

func (h *handlers) listProviderCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, newPage(providerCatalog, ""))
}
func (h *handlers) listProviderConfigs(c *gin.Context) {
	manager, ok := h.platform.(providerManager)
	if !ok {
		problem(c, 503, "cluster-unavailable", "Provider configuration discovery is unavailable", "")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	items, err := manager.ProviderConfigs(ctx)
	if err != nil {
		problem(c, 502, "cluster-error", "Could not list active provider configurations", err.Error())
		return
	}
	c.JSON(http.StatusOK, newPage(items, ""))
}

var providerVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`)

func (h *handlers) installProvider(c *gin.Context) {
	var entry *providerCatalogEntry
	for i := range providerCatalog {
		if providerCatalog[i].ID == c.Param("provider") {
			entry = &providerCatalog[i]
			break
		}
	}
	if entry == nil {
		problem(c, 404, "not-found", "Provider is not in the catalog", "")
		return
	}
	var body struct {
		Version  string   `json:"version"`
		Packages []string `json:"packages"`
	}
	if c.ShouldBindJSON(&body) != nil || !providerVersion.MatchString(body.Version) || len(body.Version) > 80 {
		problem(c, 400, "invalid-version", "Choose an explicit provider version, for example v1.2.3", "")
		return
	}
	if len(entry.Packages) > 1 && body.Version != entry.Version {
		problem(c, 400, "invalid-version", "Cloud bundles use the pinned catalog release", entry.Version)
		return
	}

	targetPackages := entry.Packages
	if len(body.Packages) > 0 {
		validMap := make(map[string]bool, len(entry.Packages))
		for _, p := range entry.Packages {
			validMap[p] = true
		}
		for _, p := range body.Packages {
			if !validMap[p] {
				problem(c, 400, "invalid-package", fmt.Sprintf("Package %q does not belong to provider %q", p, entry.ID), "")
				return
			}
		}
		targetPackages = body.Packages
		if len(entry.Packages) > 1 {
			familyPkg := entry.Packages[0]
			hasFamily := false
			for _, p := range targetPackages {
				if p == familyPkg {
					hasFamily = true
					break
				}
			}
			if !hasFamily {
				targetPackages = append([]string{familyPkg}, targetPackages...)
			}
		}
	}

	manager, ok := h.platform.(providerManager)
	if !ok {
		problem(c, 503, "cluster-unavailable", "Provider installation is unavailable", "")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	cid, _ := correlation.FromContext(ctx)
	log := slog.With("provider", entry.ID, "version", body.Version, "actor", principal(c).Identity(), "correlation.id", cid)
	log.InfoContext(ctx, "provider installation started", "stage", "checking_existing", "total", len(targetPackages))
	installed, err := h.platform.Packages(ctx, "Provider")
	if err != nil {
		log.ErrorContext(ctx, "provider installation blocked", "stage", "checking_existing", "error", err)
		problem(c, 502, "cluster-error", "Could not check installed providers", err.Error())
		return
	}
	if len(targetPackages) > 1 {
		for _, repo := range targetPackages {
			for _, p := range installed {
				if catalogPackageName(p.Package) == catalogPackageName(repo) && p.Package != repo+":"+body.Version {
					log.WarnContext(ctx, "provider installation blocked by existing package", "stage", "conflict", "existingProvider", p.Name, "existingPackage", p.Package, "requiredPackage", repo+":"+body.Version)
					problem(c, 409, "provider-conflict", "Existing provider version conflicts with this bundle", fmt.Sprintf("%s uses %s; this bundle requires %s:%s. Review the existing installation before adding packages.", p.Name, p.Package, repo, body.Version))
					return
				}
			}
		}
	}
	existing := map[string]bool{}
	for _, p := range installed {
		existing[catalogPackageName(p.Package)] = true
	}
	created, kept := 0, 0
	for _, repo := range targetPackages {
		if existing[catalogPackageName(repo)] {
			kept++
			continue
		}
		name := "crosswatc-" + catalogPackageName(repo)
		_, err := manager.InstallProvider(ctx, name, repo+":"+body.Version, principal(c).Identity())
		if err != nil {
			log.ErrorContext(ctx, "provider package creation failed", "package", repo, "created", created, "kept", kept, "total", len(targetPackages), "error", err)
		}
		if apierrors.IsAlreadyExists(err) {
			problem(c, 409, "already-installed", "A package name already exists; refresh and retry", fmt.Sprintf("%d packages added; existing packages were preserved", created))
			return
		}
		if apierrors.IsForbidden(err) {
			problem(c, 403, "cluster-forbidden", "The API service account cannot create Provider packages", fmt.Sprintf("%d packages added; existing packages were preserved", created))
			return
		}
		if err != nil {
			problem(c, 502, "cluster-error", "Provider installation incomplete", fmt.Sprintf("%d of %d packages added, %d kept. Retry to add the remaining packages. %s: %v", created, len(targetPackages), kept, repo, err))
			return
		}
		existing[catalogPackageName(repo)] = true
		created++
		log.InfoContext(ctx, "provider package added", "stage", "awaiting_crossplane", "package", repo+":"+body.Version, "created", created, "kept", kept, "total", len(targetPackages))
	}
	log.InfoContext(ctx, "provider installation submitted", "stage", "awaiting_crossplane", "created", created, "existing", kept, "total", len(targetPackages))
	c.JSON(http.StatusAccepted, gin.H{"created": created, "existing": kept, "total": len(targetPackages)})
}

func catalogPackageName(ref string) string {
	ref = strings.SplitN(ref, "@", 2)[0]
	return strings.SplitN(ref[strings.LastIndex(ref, "/")+1:], ":", 2)[0]
}

type providerObserver interface {
	ProviderDiagnostics(context.Context, string) (kube.ProviderDiagnostics, error)
	ProviderLogs(context.Context, string) ([]kube.ProviderLog, error)
}

func (h *handlers) providerDiagnostics(c *gin.Context) { h.observeProvider(c, false) }
func (h *handlers) providerLogs(c *gin.Context)        { h.observeProvider(c, true) }
func (h *handlers) observeProvider(c *gin.Context, logs bool) {
	name := c.Param("name")
	if len(validation.IsDNS1123Subdomain(name)) > 0 {
		problem(c, 400, "invalid-name", "Invalid provider name", "")
		return
	}
	reader, ok := h.platform.(providerObserver)
	if !ok {
		problem(c, 503, "cluster-unavailable", "Provider diagnostics are unavailable", "")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	var result any
	var err error
	if logs {
		result, err = reader.ProviderLogs(ctx, name)
	} else {
		result, err = reader.ProviderDiagnostics(ctx, name)
	}
	if apierrors.IsNotFound(err) {
		problem(c, 404, "not-found", "Provider not found", "")
		return
	}
	if err != nil {
		problem(c, 502, "cluster-error", "Could not read provider diagnostics", err.Error())
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, result)
}
