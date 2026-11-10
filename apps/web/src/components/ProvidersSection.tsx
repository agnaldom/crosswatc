"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { bundlePackages } from "@/lib/provider-bundles";
import { api } from "@/lib/api";
import { conditionValue } from "@/lib/conditions";
import type { ProviderCatalogEntry } from "@/lib/types";
import { Badge } from "@/components/Badge";
import { Dialog } from "@/components/Dialog";
import { ErrorState, Loading } from "@/components/States";
import { Icon } from "@/components/Icon";

function getPackageName(ref: string) {
  return ref.split("/").pop()?.split("@")[0].split(":")[0] ?? ref;
}

export function ProvidersSection({ canInstall }: { canInstall: boolean }) {
  const [view, setView] = useState<"active" | "catalog">("active");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<ProviderCatalogEntry | null>(null);
  const [selectedPackages, setSelectedPackages] = useState<string[]>([]);
  const [packageSearch, setPackageSearch] = useState("");
  const [version, setVersion] = useState("");

  const client = useQueryClient();
  const configs = useQuery({ queryKey: ["platform", "provider-configs"], queryFn: api.providerConfigs, refetchInterval: 60_000 });
  const packages = useQuery({ queryKey: ["platform", "providers"], queryFn: () => api.platformPackages("providers"), refetchInterval: selected ? 3000 : 30_000 });
  const catalog = useQuery({ queryKey: ["platform", "provider-catalog"], queryFn: api.providerCatalog, refetchInterval: false });

  const install = useMutation({
    mutationFn: () => api.installProvider(selected!.id, version, selectedPackages),
    onSuccess: () => setSelected(null),
    onSettled: () => client.invalidateQueries({ queryKey: ["platform"] }),
  });

  const query = search.trim().toLowerCase();
  const filteredConfigs = (configs.data?.items ?? []).filter(c => `${c.name} ${c.provider} ${c.apiVersion} ${c.namespace ?? ""}`.toLowerCase().includes(query));
  const filteredCatalog = (catalog.data?.items ?? []).filter(p => `${p.name} ${p.description}`.toLowerCase().includes(query));

  const installedPackageNames = new Set(
    (packages.data?.items ?? []).map(p => getPackageName(p.package))
  );

  const handleOpenInstall = (p: ProviderCatalogEntry) => {
    install.reset();
    setVersion(p.version ?? "");
    setPackageSearch("");
    if (p.packages.length > 1) {
      const already = p.packages.filter(repo => installedPackageNames.has(getPackageName(repo)));
      // Pre-select the core/family package (index 0) and any already installed packages
      setSelectedPackages(Array.from(new Set([p.packages[0], ...already])));
    } else {
      setSelectedPackages(p.packages);
    }
    setSelected(p);
  };

  const alreadyInstalledInSelected = selected
    ? selected.packages.filter(repo => installedPackageNames.has(getPackageName(repo)))
    : [];

  const filteredSelectedPackages = selected
    ? selected.packages.filter(repo => getPackageName(repo).toLowerCase().includes(packageSearch.trim().toLowerCase()))
    : [];

  const newlySelectedCount = selected
    ? selectedPackages.filter(repo => !installedPackageNames.has(getPackageName(repo))).length
    : 0;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="cw-tabs" aria-label="Provider views">
          <button
            type="button"
            aria-pressed={view === "active"}
            onClick={() => setView("active")}
            className={`cw-button ${view === "active" ? "font-semibold text-primary" : "text-on-surface-variant"}`}
          >
            Active configurations {configs.data ? `(${configs.data.items.length})` : ""}
          </button>
          <button
            type="button"
            aria-pressed={view === "catalog"}
            onClick={() => setView("catalog")}
            className={`cw-button ${view === "catalog" ? "font-semibold text-primary" : "text-on-surface-variant"}`}
          >
            Provider catalog
          </button>
        </div>
        <input
          className="cw-input max-w-xs"
          aria-label="Search providers"
          placeholder="Search providers…"
          value={search}
          onChange={e => setSearch(e.target.value)}
        />
      </div>

      {view === "active" ? (
        <section className="space-y-4" aria-label="Active provider configurations">
          <div>
            <h2 className="text-base font-semibold">Active provider configurations</h2>
            <p className="mt-1 text-sm text-on-surface-variant">Connection configurations belonging to healthy, installed providers.</p>
          </div>
          {configs.isPending ? (
            <Loading />
          ) : configs.isError ? (
            <ErrorState error={configs.error} />
          ) : filteredConfigs.length === 0 ? (
            <div className="cw-panel p-10 text-center">
              <p className="font-medium">{query ? "No configurations match your search" : "No active provider configurations"}</p>
              <p className="mx-auto mt-2 max-w-md text-sm text-on-surface-variant">
                {query ? "Try a different name or provider." : "Install a provider, configure its cloud credentials, and create a ProviderConfig to connect your account."}
              </p>
              {!query && (
                <button className="cw-button mt-5" type="button" onClick={() => setView("catalog")}>
                  Browse provider catalog
                </button>
              )}
            </div>
          ) : (
            <div className="cw-panel overflow-x-auto">
              <table className="cw-table">
                <thead>
                  <tr>
                    <th>Configuration</th>
                    <th>Provider</th>
                    <th>API</th>
                    <th>Scope</th>
                    <th>Authentication</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredConfigs.map(c => (
                    <tr key={`${c.apiVersion}/${c.kind}/${c.namespace ?? ""}/${c.name}`}>
                      <td className="font-medium">{c.name}</td>
                      <td>{c.provider}</td>
                      <td className="font-mono text-xs">{c.apiVersion}</td>
                      <td>{c.namespace || "Cluster"}</td>
                      <td>{c.credentialSource || "Provider-specific"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      ) : (
        <section className="space-y-4" aria-label="Available provider catalog">
          <div>
            <h2 className="text-base font-semibold">Provider catalog</h2>
            <p className="mt-1 text-sm text-on-surface-variant">Add a provider to your control plane, then configure access to your account.</p>
          </div>
          {!canInstall && <p className="text-sm text-on-surface-variant">An administrator can install providers. Your access is read-only.</p>}
          {packages.isError && <ErrorState error={packages.error} />}
          {catalog.isPending ? (
            <Loading />
          ) : catalog.isError ? (
            <ErrorState error={catalog.error} />
          ) : filteredCatalog.length === 0 ? (
            <p className="text-sm text-on-surface-variant">No providers match your search.</p>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {filteredCatalog.map(p => {
                const installed = bundlePackages(p.packages, packages.data?.items ?? []);
                const complete = installed.length === p.packages.length;
                const ready = complete && installed.every(item => conditionValue(item.conditions, "Healthy") === true && conditionValue(item.conditions, "Installed") === true);
                const failed = installed.some(item => conditionValue(item.conditions, "Healthy") === false || conditionValue(item.conditions, "Installed") === false);
                return (
                  <article key={p.id} className="cw-panel flex flex-col gap-4 p-5">
                    <div className="flex items-start justify-between gap-3">
                      <span className="grid h-10 w-10 place-items-center rounded-xl bg-surface-container text-primary">
                        <Icon name="deployed_code" className="h-5 w-5" aria-hidden />
                      </span>
                      {installed.length > 0 && (
                        <Badge tone={ready ? "ok" : failed ? "error" : "warn"}>
                          {ready ? "Installed" : failed ? "Needs attention" : complete ? "Installing" : `${installed.length}/${p.packages.length} added`}
                        </Badge>
                      )}
                    </div>
                    <div>
                      <h3 className="font-semibold">{p.name}</h3>
                      <p className="mt-2 text-sm leading-6 text-on-surface-variant">{p.description}</p>
                    </div>
                    <p className="text-xs text-on-surface-variant">
                      {p.packages.length} {p.packages.length === 1 ? "package" : "packages · complete cloud bundle"}
                    </p>
                    <p className="text-xs text-on-surface-variant">
                      {installed.filter(item => conditionValue(item.conditions, "Healthy") === true && conditionValue(item.conditions, "Installed") === true).length} ready · {installed.length} added · {p.packages.length - installed.length} missing
                    </p>
                    {failed && (
                      <p className="text-xs text-on-surface-variant">
                        One or more packages need attention. Open installation details to see the reason, events, and controller logs.
                      </p>
                    )}
                    <Link href={`/providers/${p.id}/installation`} className="cw-button justify-center">
                      View installation &amp; logs
                    </Link>
                    <div className="mt-auto flex items-center justify-between gap-3 pt-2">
                      <a className="text-sm text-primary hover:underline" href={p.documentation} target="_blank" rel="noreferrer">
                        Documentation ↗
                      </a>
                      <button
                        type="button"
                        className="cw-button"
                        disabled={!canInstall || complete || !packages.data || packages.isError}
                        onClick={() => handleOpenInstall(p)}
                      >
                        {complete ? "All packages added" : installed.length ? "Add packages" : "Install provider"}
                      </button>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </section>
      )}

      {selected && (
        <Dialog title={`Install ${selected.name}`} busy={install.isPending} onClose={() => setSelected(null)}>
          <form
            className="space-y-5"
            onSubmit={e => {
              e.preventDefault();
              install.mutate();
            }}
          >
            <p className="text-sm leading-6 text-on-surface-variant">
              {selected.packages.length > 1
                ? `Select the service packages to install on your cluster. Shared authentication (Core) is automatically included.`
                : `This adds the provider package to the connected cluster. Existing packages are preserved.`}
            </p>

            {selected.packages.length > 1 && (
              <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 p-3 text-xs text-on-surface-variant">
                <span className="font-semibold text-amber-600 dark:text-amber-400">Resource tip:</span> Installing all {selected.packages.length} controllers at once requires substantial memory and CPU. For local environments, select only the services you need (e.g. compute, storage, sql).
              </div>
            )}

            {selected.packages.length > 1 && (
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <label htmlFor="package-search-input" className="text-xs font-semibold uppercase tracking-wider text-on-surface-variant">
                    Packages ({selectedPackages.length} of {selected.packages.length} selected)
                  </label>
                  <div className="flex gap-2 text-xs">
                    <button
                      type="button"
                      className="text-primary hover:underline"
                      onClick={() => setSelectedPackages(selected.packages)}
                      disabled={install.isPending}
                    >
                      Select all
                    </button>
                    <span className="text-outline-variant">·</span>
                    <button
                      type="button"
                      className="text-on-surface-variant hover:underline"
                      onClick={() => setSelectedPackages(Array.from(new Set([selected.packages[0], ...alreadyInstalledInSelected])))}
                      disabled={install.isPending}
                    >
                      Clear optional
                    </button>
                  </div>
                </div>

                <input
                  id="package-search-input"
                  className="cw-input w-full text-xs"
                  placeholder="Filter packages (e.g. storage, compute, sql)…"
                  value={packageSearch}
                  onChange={e => setPackageSearch(e.target.value)}
                />

                <div className="max-h-56 overflow-y-auto space-y-1 rounded-lg border border-outline-variant/40 p-2 bg-surface-container-lowest text-xs">
                  {filteredSelectedPackages.length === 0 ? (
                    <p className="p-3 text-center text-on-surface-variant">No packages match &quot;{packageSearch}&quot;.</p>
                  ) : (
                    filteredSelectedPackages.map(repo => {
                      const name = getPackageName(repo);
                      const isFamily = repo === selected.packages[0];
                      const isInstalled = alreadyInstalledInSelected.includes(repo);
                      const isChecked = selectedPackages.includes(repo) || isFamily || isInstalled;

                      return (
                        <label
                          key={repo}
                          className={`flex items-center justify-between gap-2 p-1.5 rounded cursor-pointer transition-colors hover:bg-surface-container-high ${
                            isFamily ? "bg-surface-container/60" : ""
                          }`}
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            <input
                              type="checkbox"
                              className="rounded border-outline-variant text-primary focus:ring-primary"
                              disabled={isFamily || isInstalled || install.isPending}
                              checked={isChecked}
                              onChange={e => {
                                if (e.target.checked) {
                                  setSelectedPackages(prev => Array.from(new Set([...prev, repo])));
                                } else {
                                  setSelectedPackages(prev => prev.filter(r => r !== repo));
                                }
                              }}
                            />
                            <span className="font-mono text-xs truncate" title={repo}>
                              {name}
                            </span>
                          </div>
                          {isFamily ? (
                            <Badge tone="warn">Required (Core)</Badge>
                          ) : isInstalled ? (
                            <Badge tone="ok">Already added</Badge>
                          ) : null}
                        </label>
                      );
                    })
                  )}
                </div>
              </div>
            )}

            <div>
              <label htmlFor="provider-version" className="mb-2 block text-sm font-medium">
                Package version
              </label>
              <input
                id="provider-version"
                className="cw-input w-full"
                readOnly={selected.packages.length > 1}
                value={version}
                onChange={e => setVersion(e.target.value)}
                placeholder="v1.2.3"
                required
                pattern="v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?"
              />
              <p className="mt-2 break-all font-mono text-xs text-on-surface-variant">
                {selected.packages[0]}:{version || "<version>"}
              </p>
            </div>

            {install.isPending && (
              <div className="space-y-2" role="status">
                <p className="text-sm">
                  {bundlePackages(selectedPackages, packages.data?.items ?? []).length} / {selectedPackages.length} packages added. Crossplane will finish installing the controllers.
                </p>
                <progress className="w-full" max={selectedPackages.length} value={bundlePackages(selectedPackages, packages.data?.items ?? []).length} />
              </div>
            )}

            {install.isError && <ErrorState error={install.error} />}

            <div className="flex justify-end gap-3">
              <button type="button" className="cw-button" disabled={install.isPending} onClick={() => setSelected(null)}>
                Cancel
              </button>
              <button
                type="submit"
                className="cw-button font-medium"
                disabled={
                  install.isPending ||
                  selectedPackages.length === 0 ||
                  (alreadyInstalledInSelected.length > 0 && newlySelectedCount === 0) ||
                  !/^v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?$/.test(version)
                }
              >
                {install.isPending
                  ? "Requesting installation…"
                  : alreadyInstalledInSelected.length > 0 && newlySelectedCount === 0
                  ? "Selected packages already added"
                  : selected.packages.length > 1
                  ? `Install ${newlySelectedCount} package${newlySelectedCount === 1 ? "" : "s"}`
                  : "Install"}
              </button>
            </div>
          </form>
        </Dialog>
      )}
    </div>
  );
}
