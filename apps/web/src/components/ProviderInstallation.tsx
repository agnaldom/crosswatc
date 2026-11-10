"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { api } from "@/lib/api";
import { bundlePackages, providerState } from "@/lib/provider-bundles";
import { Badge } from "@/components/Badge";
import { PageHeading } from "@/components/PageHeading";
import { ErrorState, Loading } from "@/components/States";

export function ProviderInstallation({ providerID }: { providerID: string }) {
  const [selected, setSelected] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [showLogs, setShowLogs] = useState(false);
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, refetchInterval: false });
  const catalog = useQuery({ queryKey: ["platform", "provider-catalog"], queryFn: api.providerCatalog, refetchInterval: false });
  const packages = useQuery({ queryKey: ["platform", "providers"], queryFn: () => api.platformPackages("providers"), refetchInterval: 5000 });
  const entry = catalog.data?.items.find((p) => p.id === providerID);
  const installed = entry ? bundlePackages(entry.packages, packages.data?.items ?? []) : [];
  // Include duplicate publishers/versions so dependency conflicts remain visible.
  const candidates = (packages.data?.items ?? []).filter((p) => entry && bundlePackages(entry.packages, [p]).length > 0);
  const rank = (p: (typeof candidates)[number]) => providerState(p).failed ? 0 : providerState(p).ready ? 2 : 1;
  const ordered = [...candidates].sort((a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name));
  const conflicts = entry && entry.packages.length > 1 ? candidates.filter((p) => entry.packages.some((repo) => bundlePackages([repo], [p]).length > 0 && p.package !== `${repo}:${entry.version}`)) : [];
  const name = selected ?? ordered[0]?.name;
  const active = candidates.find((p) => p.name === name);
  const details = useQuery({ queryKey: ["platform", "provider-diagnostics", name], queryFn: () => api.providerDiagnostics(name!), enabled: !!active, refetchInterval: 5000, retry: false });
  const logs = useQuery({ queryKey: ["platform", "provider-logs", name], queryFn: () => api.providerLogs(name!), enabled: !!active && showLogs && !!me.data?.admin, refetchInterval: showLogs ? 5000 : false, retry: false });
  const ready = installed.filter((p) => providerState(p).ready).length;
  const failed = installed.filter((p) => providerState(p).failed).length;
  const currentProvider = details.data?.provider ?? active;
  const conditions = [...(currentProvider?.conditions ?? []).map((c) => ({ ...c, source: "Provider" })), ...(details.data?.revisionConditions ?? []).map((c) => ({ ...c, source: "Revision" }))];

  if (catalog.isPending || packages.isPending) return <Loading label="Loading installation status…" />;
  if (catalog.isError) return <ErrorState error={catalog.error} />;
  if (packages.isError && !packages.data) return <ErrorState error={packages.error} />;
  if (!entry) return <p>Provider not found in the catalog.</p>;

  return <>
    <PageHeading eyebrow="Providers" title={`${entry.name} installation`} description="Live package status, dependency errors, Kubernetes events and controller logs." actions={<Link className="cw-button" href="/providers">Back to providers</Link>} />
    {conflicts.length > 0 && <section className="cw-panel space-y-2 border-amber-300 p-5" aria-label="Conflicting provider versions"><h2 className="font-semibold">Existing installation conflicts with this bundle</h2><p className="text-sm text-on-surface-variant">Some provider objects use a different version or publisher than this bundle ({entry.version}). Dependencies and CRD ownership may prevent the new controllers from becoming ready. Review these objects before changing the existing installation.</p>{conflicts.map((p) => <p key={p.name} className="break-all font-mono text-xs">{p.name}: {p.package}</p>)}</section>}
    {packages.isError && <div className="cw-panel space-y-2 p-4"><p className="text-sm">Showing the last successful snapshot. The cluster could not be refreshed.</p><ErrorState error={packages.error} /></div>}
    <div className="grid gap-3 sm:grid-cols-4" aria-label="Installation progress">
      {[ ["Ready", ready], ["Needs attention", failed], ["Waiting / installing", installed.length - ready - failed], ["Not added", entry.packages.length - installed.length] ].map(([label, value]) => <div key={label} className="cw-panel p-4"><p className="text-xs text-on-surface-variant">{label}</p><p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p></div>)}
    </div>
    <div className="flex flex-wrap items-center justify-between gap-3 text-xs text-on-surface-variant"><span>{installed.length} / {entry.packages.length} packages added · creating a package does not mean its controller is ready.</span><span role="status">Updated {new Date(packages.dataUpdatedAt).toLocaleTimeString()} · refreshes every 5s</span></div>
    <div className="grid items-start gap-5 xl:grid-cols-2">
      <section className="cw-panel overflow-hidden" aria-label="Provider packages">
        <div className="border-b border-outline-variant p-4"><h2 className="font-semibold">Packages</h2><input className="cw-input mt-3 w-full" aria-label="Search installation packages" placeholder="Filter by package, version or error…" value={search} onChange={(e) => setSearch(e.target.value)} /></div>
        <div className="max-h-[36rem] overflow-auto"><table className="cw-table"><thead><tr><th>Package / provider object</th><th>Status</th></tr></thead><tbody>
          {ordered.filter((p) => `${p.name} ${p.package} ${p.conditions?.map((c) => c.message ?? "").join(" ")}`.toLowerCase().includes(search.toLowerCase())).map((p) => {
            const state = providerState(p);
            const message = p.conditions?.find((c) => c.status !== "True" && (c.message || c.reason));
            return <tr key={p.name} aria-selected={p.name === name} className={p.name === name ? "bg-primary/5" : undefined}>
              <td><button type="button" className="text-left font-medium text-primary hover:underline" onClick={() => setSelected(p.name)}>{p.name}</button><p className="mt-1 break-all font-mono text-[11px] text-on-surface-variant">{p.package}</p>{message && <p className="mt-2 max-w-md break-words text-xs text-on-surface-variant">{message.message || message.reason}</p>}</td>
              <td><Badge tone={state.tone}>{state.label}</Badge></td>
            </tr>;
          })}
        </tbody></table>{!ordered.length && <p className="p-6 text-sm text-on-surface-variant">No package objects have been added yet.</p>}</div>
      </section>
      <section className="space-y-5" aria-label="Selected package diagnostics">
        {active ? <>
          <div className="cw-panel space-y-4 p-5"><h2 className="break-all font-semibold">{active.name}</h2><p className="break-all font-mono text-xs text-on-surface-variant">Revision: {currentProvider?.currentRevision || "Not created yet"}</p>
            <h3 className="text-sm font-semibold">Conditions and errors</h3>
            {conditions.length ? conditions.map((c, i) => <div key={`${c.source}-${c.type}-${i}`} className="rounded-lg border border-outline-variant p-3"><div className="flex flex-wrap items-center gap-2 text-xs"><span>{c.source} · {c.type}</span><Badge tone={c.status === "True" ? "ok" : c.status === "False" ? "error" : "warn"}>{c.status}</Badge></div><p className="mt-2 text-sm font-medium">{c.reason || "Waiting for condition"}</p>{c.message && <p className="mt-1 whitespace-pre-wrap break-words text-xs leading-5 text-on-surface-variant">{c.message}</p>}</div>) : <p className="text-sm text-on-surface-variant">Crossplane has not reported conditions yet.</p>}
            {details.isFetching && <p className="text-xs text-on-surface-variant">Refreshing diagnostics…</p>}{details.isError && <ErrorState error={details.error} />}
            {details.data?.warnings.map((warning, i) => <p key={i} className="break-words text-xs text-amber-700 dark:text-amber-400">{warning}</p>)}
          </div>
          <div className="cw-panel space-y-3 p-5"><h3 className="text-sm font-semibold">Runtime pods</h3>
            {details.data?.pods.map((pod) => <div key={`${pod.namespace}/${pod.name}`} className="space-y-2 rounded-lg border border-outline-variant p-3"><p className="break-all text-xs font-medium">{pod.namespace}/{pod.name} · {pod.phase}</p>{pod.containers.map((c) => <p key={c.name} className="break-words text-xs text-on-surface-variant">{c.name}: {c.state} · {c.restarts} restarts {c.reason ? `· ${c.reason}` : ""}{c.message ? ` — ${c.message}` : ""}</p>)}</div>)}
            {details.data && !details.data.pods.length && <p className="text-xs text-on-surface-variant">No runtime pod found for this revision in crossplane-system. Check conditions and events: the package may still be downloading or waiting for dependencies.</p>}
          </div>
          {details.data?.controllerPods && details.data.controllerPods.length > 0 && <div className="cw-panel space-y-3 p-5"><h3 className="text-sm font-semibold">Crossplane controller health</h3>{details.data.controllerPods.map((pod) => <div key={pod.name} className="text-xs"><p className="break-all font-medium">{pod.name} · {pod.phase}</p>{pod.containers.map((c) => <p key={c.name} className="mt-1 text-on-surface-variant">{c.name}: {c.state} · {c.restarts} restarts {c.reason ? `· ${c.reason}` : ""}</p>)}</div>)}</div>}
          <div className="cw-panel space-y-3 p-5"><h3 className="text-sm font-semibold">Installation events</h3>
            {details.data?.events.map((event, i) => <div key={i} className="border-b border-outline-variant pb-3 text-xs"><div className="flex flex-wrap justify-between gap-2"><span className="font-medium">{event.type} · {event.reason}</span><span className="text-on-surface-variant">{event.lastSeen ? new Date(event.lastSeen).toLocaleTimeString() : ""} · ×{event.count}</span></div><p className="mt-1 break-words leading-5 text-on-surface-variant">{event.message}</p></div>)}
            {details.data && !details.data.events.length && <p className="text-xs text-on-surface-variant">No retained events for this package or revision. Kubernetes events expire.</p>}
          </div>
          <div className="cw-panel space-y-4 p-5"><div className="flex flex-wrap items-center justify-between gap-3"><h3 className="text-sm font-semibold">Controller logs</h3><button className="cw-button" type="button" disabled={!me.data?.admin} aria-pressed={showLogs} onClick={() => setShowLogs(!showLogs)}>{showLogs ? "Pause logs" : "Show logs"}</button></div>
            {!me.data?.admin && <p className="text-xs text-on-surface-variant">Controller logs require administrator access.</p>}
            {showLogs && <><p className="text-xs text-on-surface-variant">Last 200 lines per container in crossplane-system. Crossplane output includes this package and controller-wide failures. Previous container logs are included after restarts.</p>{logs.isPending && <Loading label="Reading controller logs…" />}{logs.isError && <ErrorState error={logs.error} />}{logs.data?.map((log, i) => <div key={i} className="space-y-2"><p className="break-all text-xs font-medium">{log.source} · {log.pod || "unavailable"} / {log.container}{log.previous ? " · previous container" : ""}</p>{log.error ? <p className="break-words text-xs text-amber-700 dark:text-amber-400">{log.error}</p> : <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-slate-950 p-4 text-xs leading-5 text-slate-100">{log.text || "No matching log lines in the retained tail. Check the conditions and events above."}</pre>}</div>)}{logs.data?.length === 0 && <p className="text-xs text-on-surface-variant">No provider or Crossplane controller pods were found in crossplane-system.</p>}</>}
          </div>
        </> : <p className="cw-panel p-6 text-sm text-on-surface-variant">Select a package to inspect its installation.</p>}
      </section>
    </div>
  </>;
}
