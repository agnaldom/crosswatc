import { ApiError } from "@/lib/api";

export function Loading({ label = "Carregando…" }: { label?: string }) {
  return <p className="py-8 text-sm text-zinc-500">{label}</p>;
}

export function ErrorState({ error }: { error: unknown }) {
  const e = error instanceof ApiError ? error : null;
  return (
    <div className="rounded-lg border border-rose-200 bg-rose-50 p-4 text-sm text-rose-800" role="alert">
      <p className="font-medium">
        {e?.status === 404 ? "Recurso não encontrado" : e && e.status >= 400 && e.status < 500 ? e.message : "Não foi possível concluir a operação"}
      </p>
      {e?.correlationId && <p className="mt-1 font-mono text-xs">correlation id: {e.correlationId}</p>}
    </div>
  );
}

export function Empty({ children }: { children: React.ReactNode }) {
  return <p className="rounded-lg border border-dashed border-zinc-300 p-6 text-center text-sm text-zinc-500">{children}</p>;
}
