"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Badge } from "@/components/Badge";
import { Empty, ErrorState, Loading } from "@/components/States";
import { api } from "@/lib/api";
import { normalizeConditions } from "@/lib/conditions";
import type { CrossplanePackage } from "@/lib/types";

const tabs = ["Providers", "Functions", "Configurations", "XRDs", "Compositions"] as const;
type Tab = (typeof tabs)[number];

export default function PlatformPage() {
  const [tab, setTab] = useState<Tab>("Providers");
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, refetchInterval: false });
  if (me.isPending) return <Loading />;
  if (me.isError) return <ErrorState error={me.error} />;
  if (!me.data.sre) {
    return <p className="rounded-md bg-amber-50 p-4 text-sm text-amber-900">O catálogo de plataforma é restrito ao time de SRE/Plataforma.</p>;
  }
  return (
    <div className="mx-auto max-w-6xl">
      <h1 className="text-2xl font-semibold tracking-tight">Platform · Crossplane</h1>
      <p className="mt-1 text-sm text-zinc-600">Saúde da instalação do Crossplane no cluster de controle (somente leitura).</p>
      <nav className="mt-6 flex gap-1 border-b border-zinc-200" role="tablist">
        {tabs.map((t) => (
          <button key={t} role="tab" aria-selected={tab === t} type="button" onClick={() => setTab(t)}
            className={`-mb-px border-b-2 px-3 py-2 text-sm ${tab === t ? "border-zinc-900 font-medium" : "border-transparent text-zinc-500 hover:text-zinc-800"}`}>
            {t}
          </button>
        ))}
      </nav>
      <section className="mt-5">
        {tab === "Providers" && <Packages kind="providers" />}
        {tab === "Functions" && <Packages kind="functions" />}
        {tab === "Configurations" && <Packages kind="configurations" />}
        {tab === "XRDs" && <XRDs />}
        {tab === "Compositions" && <Compositions />}
      </section>
    </div>
  );
}

function Table({ head, children }: { head: string[]; children: React.ReactNode }) {
  return (
    <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
      <table className="w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase tracking-wide text-zinc-500">
          <tr>{head.map((h) => <th key={h} className="px-4 py-2 font-medium">{h}</th>)}</tr>
        </thead>
        <tbody className="divide-y divide-zinc-100">{children}</tbody>
      </table>
    </div>
  );
}

function PackageRow({ p }: { p: CrossplanePackage }) {
  const st = normalizeConditions(p.conditions, p.kind);
  return (
    <tr className="align-top">
      <td className="px-4 py-2.5 font-medium">{p.name}</td>
      <td className="px-4 py-2.5" title={st.detail}>
        <Badge tone={st.tone}>{st.label}</Badge>
        {st.detail && <p className="mt-1 max-w-xs text-xs text-zinc-500">{st.detail}</p>}
      </td>
      <td className="break-all px-4 py-2.5 font-mono text-xs">{p.package}</td>
      <td className="px-4 py-2.5 font-mono text-xs text-zinc-600">{p.currentRevision ?? "—"}</td>
      <td className="px-4 py-2.5 text-xs text-zinc-600">{p.revisionActivationPolicy ?? "Automatic"}</td>
    </tr>
  );
}

function Packages({ kind }: { kind: "providers" | "functions" | "configurations" }) {
  const q = useQuery({ queryKey: ["platform", kind], queryFn: () => api.platformPackages(kind) });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhum pacote instalado.</Empty>;
  const unhealthy = q.data.items.filter((p) => normalizeConditions(p.conditions, p.kind).tone === "error").length;
  return (
    <div className="space-y-2">
      <p className="text-sm text-zinc-600">{q.data.items.length} instalado(s){unhealthy ? `, ${unhealthy} com problema` : ""}.</p>
      <Table head={["Nome", "Status", "Pacote", "Revisão atual", "Ativação"]}>
        {q.data.items.map((p) => <PackageRow key={p.name} p={p} />)}
      </Table>
    </div>
  );
}

function XRDs() {
  const q = useQuery({ queryKey: ["platform", "xrds"], queryFn: api.platformXRDs });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhum XRD.</Empty>;
  return (
    <Table head={["XRD", "Status", "Kind (XR)", "Claim", "Escopo", "Versão"]}>
      {q.data.items.map((x) => (
        <tr key={x.name}>
          <td className="px-4 py-2.5 font-mono text-xs">{x.name}</td>
          <td className="px-4 py-2.5"><Badge tone={x.established ? "ok" : "error"}>{x.established ? "Established" : "Not Established"}</Badge></td>
          <td className="px-4 py-2.5">{x.kind}</td>
          <td className="px-4 py-2.5">{x.claimKind ?? "—"}</td>
          <td className="px-4 py-2.5 text-zinc-600">{x.scope}</td>
          <td className="px-4 py-2.5 font-mono text-xs">{x.version}</td>
        </tr>
      ))}
    </Table>
  );
}

function Compositions() {
  const q = useQuery({ queryKey: ["platform", "compositions"], queryFn: api.platformCompositions });
  if (q.isPending) return <Loading />;
  if (q.isError) return <ErrorState error={q.error} />;
  if (!q.data.items.length) return <Empty>Nenhuma Composition.</Empty>;
  return (
    <Table head={["Composition", "XR", "Modo", "Steps", "Functions", "Última revisão"]}>
      {q.data.items.map((c) => (
        <tr key={c.name} className="align-top">
          <td className="px-4 py-2.5 font-medium">{c.name}</td>
          <td className="px-4 py-2.5 font-mono text-xs">{c.compositeType}</td>
          <td className="px-4 py-2.5"><Badge tone={c.mode === "Pipeline" ? "ok" : "warn"}>{c.mode}</Badge></td>
          <td className="px-4 py-2.5">{c.steps}</td>
          <td className="px-4 py-2.5 font-mono text-xs">{c.functions?.join(", ") || "—"}</td>
          <td className="px-4 py-2.5 font-mono text-xs text-zinc-600">{c.latestRevision ?? "—"}</td>
        </tr>
      ))}
    </Table>
  );
}
