"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { use, useState } from "react";
import { ErrorState, Loading } from "@/components/States";
import { api } from "@/lib/api";

export default function DecommissionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const resource = useQuery({ queryKey: ["resource", id], queryFn: () => api.resource(id) });
  const preview = useQuery({ queryKey: ["decommission", id], queryFn: () => api.decommissionPreview(id), refetchInterval: false });
  const [confirm, setConfirm] = useState("");
  const [justification, setJustification] = useState("");
  const qc = useQueryClient();
  const submit = useMutation({
    mutationFn: () => api.decommission(id, confirm, justification),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["state", id] }),
  });

  if (resource.isPending || preview.isPending) return <Loading />;
  if (resource.isError) return <ErrorState error={resource.error} />;
  if (preview.isError) return <ErrorState error={preview.error} />;
  const p = preview.data;
  const ready = p.allowed && confirm === p.confirmName && justification.trim().length >= 15;

  return (
    <div className="mx-auto max-w-3xl">
      <Link href={`/infrastructure/${id}`} className="text-sm text-zinc-500 hover:underline">← {resource.data.name}</Link>
      <h1 className="mt-3 text-2xl font-semibold tracking-tight text-rose-800">Decommission</h1>
      <p className="mt-1 text-sm text-zinc-600">
        Opens a Pull Request removing the Claim. After approval and merge, Argo CD prunes the resources and Crossplane deletes the infrastructure.
      </p>

      {!p.allowed && <p className="mt-4 rounded-md bg-amber-50 p-3 text-sm text-amber-900" role="alert">{p.reason}</p>}

      <section className="mt-6 rounded-lg border border-rose-200 bg-white p-5 text-sm">
        <h2 className="font-semibold">Impact</h2>
        <p className="mt-2">Will be <strong>deleted</strong> in cloud ({p.deleted.length}):</p>
        <ul className="mt-1 list-disc pl-5 font-mono text-xs">
          {p.deleted.map((d) => <li key={d.kind + d.name}>{d.kind}/{d.name}{d.externalName ? ` (${d.externalName})` : ""}</li>)}
        </ul>
        {p.orphaned.length > 0 && (
          <>
            <p className="mt-3">Retained (<span className="font-mono">deletionPolicy: Orphan</span>):</p>
            <ul className="mt-1 list-disc pl-5 font-mono text-xs">
              {p.orphaned.map((d) => <li key={d.kind + d.name}>{d.kind}/{d.name}</li>)}
            </ul>
          </>
        )}
        {p.warnings.length > 0 && (
          <ul className="mt-3 space-y-1 rounded-md bg-amber-50 p-3 text-amber-900">{p.warnings.map((w) => <li key={w}>{w}</li>)}</ul>
        )}
      </section>

      {p.allowed && !submit.data && (
        <section className="mt-6 space-y-4 rounded-lg border border-zinc-200 bg-white p-5 text-sm">
          <div>
            <label htmlFor="confirm" className="block font-medium">
              Type <span className="font-mono">{p.confirmName}</span> to confirm
            </label>
            <input id="confirm" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="off"
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-1.5 font-mono" />
          </div>
          <div>
            <label htmlFor="why" className="block font-medium">Justification (min. 15 characters)</label>
            <textarea id="why" rows={3} value={justification} onChange={(e) => setJustification(e.target.value)}
              className="mt-1 w-full rounded-md border border-zinc-300 px-3 py-2" />
          </div>
          <button type="button" disabled={!ready || submit.isPending}
            onClick={() => window.confirm(`Open PR to remove ${p.confirmName}? This action will delete cloud resources after merge.`) && submit.mutate()}
            className="rounded-md bg-rose-700 px-4 py-2 font-medium text-white hover:bg-rose-600 disabled:opacity-40">
            {submit.isPending ? "Opening Pull Request…" : "Open decommission PR"}
          </button>
          {submit.isError && <ErrorState error={submit.error} />}
        </section>
      )}

      {submit.data && (
        <div className="mt-6 rounded-md bg-emerald-50 p-4 text-sm text-emerald-900" role="status">
          Pull Request <a className="underline" href={submit.data.prUrl} target="_blank" rel="noreferrer">#{submit.data.prNumber}</a> opened
          (revision {submit.data.rev}). Track in <Link className="underline" href={`/infrastructure/${id}`}>Lifecycle</Link>.
        </div>
      )}
    </div>
  );
}
