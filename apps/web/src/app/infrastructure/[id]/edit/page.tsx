"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { use, useState } from "react";
import { DynamicForm } from "@/components/DynamicForm";
import { PreviewPanel } from "@/components/PreviewPanel";
import { Empty, ErrorState, Loading } from "@/components/States";
import { ApiError, api } from "@/lib/api";
import { changedValues } from "@/lib/form-schema";

export default function EditPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const resource = useQuery({ queryKey: ["resource", id], queryFn: () => api.resource(id) });
  const form = useQuery({ queryKey: ["form", id], queryFn: () => api.form(id), refetchInterval: false });
  const [changes, setChanges] = useState<Record<string, unknown> | null>(null);
  const preview = useMutation({ mutationFn: (c: Record<string, unknown>) => api.preview(id, c) });
  const [justification, setJustification] = useState("");
  const router = useRouter();
  const qc = useQueryClient();
  const submit = useMutation({
    mutationFn: (supersede: boolean) => api.createChange(id, changes ?? {}, justification, supersede),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["state", id] });
      qc.invalidateQueries({ queryKey: ["revisions", id] });
    },
  });

  if (resource.isPending || form.isPending) return <Loading />;
  if (resource.isError) return <ErrorState error={resource.error} />;
  if (form.isError) return <ErrorState error={form.error} />;
  const { form: schema, values, editable, reason } = form.data;

  return (
    <div className="mx-auto max-w-3xl">
      <Link href={`/infrastructure/${id}`} className="text-sm text-zinc-500 hover:underline">← {resource.data.name}</Link>
      <h1 className="mt-3 text-2xl font-semibold tracking-tight">Edit Configuration</h1>
      <p className="mt-1 text-sm text-zinc-600">
        Somente campos autorizados são editáveis. Toda mudança vira um Pull Request.
      </p>
      {!editable && <p className="mt-4 rounded-md bg-amber-50 p-3 text-sm text-amber-800">{reason}</p>}
      <div className="mt-6 rounded-lg border border-zinc-200 bg-white p-6">
        {schema.fields.length === 0 ? (
          <Empty>Este tipo de recurso não expõe parâmetros.</Empty>
        ) : (
          <DynamicForm fields={editable ? schema.fields : schema.fields.map((f) => ({ ...f, editable: false }))}
            values={values} submitLabel="Revisar mudanças"
            busy={preview.isPending}
            onSubmit={(v) => {
              const c = changedValues(schema.fields, values, v);
              setChanges(c);
              if (Object.keys(c).length) preview.mutate(c);
            }} />
        )}
      </div>
      {changes && Object.keys(changes).length === 0 && (
        <p className="mt-6 rounded-lg border border-zinc-200 bg-white p-4 text-sm text-zinc-600">Nenhuma alteração.</p>
      )}
      {preview.isError && <div className="mt-6"><ErrorState error={preview.error} /></div>}
      {preview.data && changes && Object.keys(changes).length > 0 && (
        <section className="mt-6 rounded-lg border border-zinc-200 bg-white p-6">
          <h2 className="mb-4 text-lg font-semibold">Preview</h2>
          <PreviewPanel preview={preview.data} />
          {preview.data.valid && !submit.data && (
            <div className="mt-6 space-y-3">
              <label htmlFor="justification" className="block text-sm font-medium">Justificativa *</label>
              <textarea id="justification" rows={3} value={justification} onChange={(e) => setJustification(e.target.value)}
                className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm" placeholder="Por que esta mudança é necessária?" />
              <button type="button" disabled={!justification.trim() || submit.isPending} onClick={() => submit.mutate(false)}
                className="rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-600 disabled:opacity-50">
                {submit.isPending ? "Abrindo Pull Request…" : "Confirmar e abrir Pull Request"}
              </button>
              {submit.isError && <ErrorState error={submit.error} />}
              {submit.error instanceof ApiError && submit.error.status === 409 && submit.error.message.startsWith("Revision already open") && (
                <button type="button" onClick={() => submit.mutate(true)} className="text-sm text-amber-800 underline">
                  Substituir a revision aberta (fecha o Pull Request anterior)
                </button>
              )}
            </div>
          )}
          {submit.data && (
            <div className="mt-6 rounded-md bg-emerald-50 p-4 text-sm text-emerald-900" role="status">
              <p className="font-medium">Revision {submit.data.rev} criada.</p>
              <p className="mt-1">
                Pull Request <a className="underline" href={submit.data.prUrl} target="_blank" rel="noreferrer">#{submit.data.prNumber}</a> aberto.
                Correlation ID <span className="font-mono text-xs">{submit.data.correlationId}</span>.
              </p>
              <button type="button" onClick={() => router.push(`/infrastructure/${id}`)} className="mt-3 underline">
                Acompanhar o lifecycle
              </button>
            </div>
          )}
        </section>
      )}
    </div>
  );
}
