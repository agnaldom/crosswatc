import { Badge } from "@/components/Badge";
import { actionOwnerLabel, stageLabel, stageTone } from "@/lib/labels";
import type { StageView } from "@/lib/types";

const icon: Record<StageView["status"], string> = {
  SUCCEEDED: "✓",
  FAILED: "✕",
  IN_PROGRESS: "●",
  PENDING: "○",
  SKIPPED: "–",
};

export function Lifecycle({ stages }: { stages: StageView[] }) {
  if (!stages.length) return <p className="text-sm text-zinc-500">Aguardando primeira reconciliação.</p>;
  return (
    <ol className="space-y-3">
      {stages.map((s) => (
        <li key={s.stage} className="flex gap-3">
          <span aria-hidden className={`mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold ${
            s.status === "SUCCEEDED" ? "bg-emerald-100 text-emerald-700"
              : s.status === "FAILED" ? "bg-rose-100 text-rose-700"
              : s.status === "IN_PROGRESS" ? "bg-sky-100 text-sky-700"
              : "bg-zinc-100 text-zinc-500"}`}>
            {icon[s.status]}
          </span>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium">{stageLabel[s.stage]}</span>
              <Badge tone={stageTone[s.status]}>{s.status.replace("_", " ").toLowerCase()}</Badge>
              {s.actionOwner && s.status !== "SUCCEEDED" && (
                <span className="text-xs text-zinc-600">Quem age: <strong>{actionOwnerLabel[s.actionOwner]}</strong></span>
              )}
            </div>
            {s.message && <p className="text-sm text-zinc-600">{s.message}</p>}
            {s.technicalDetail && (
              <details className="mt-1 text-xs text-zinc-500">
                <summary className="cursor-pointer select-none">Detalhes técnicos</summary>
                <code className="mt-1 block whitespace-pre-wrap rounded bg-zinc-100 p-2">{s.technicalDetail}</code>
              </details>
            )}
          </div>
        </li>
      ))}
    </ol>
  );
}
