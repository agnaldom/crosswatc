import type { Condition } from "@/lib/conditions";

// One card per status.condition, as in Crossview's "Status Conditions" panel.
export function ConditionCards({ conditions, compact = false }: { conditions?: Condition[]; compact?: boolean }) {
  if (!conditions?.length) return <p className="text-xs text-zinc-500">Sem condições reportadas.</p>;
  return (
    <div className="space-y-2">
      {conditions.map((c) => {
        const tone = c.status === "True" ? "ok" : c.status === "False" ? "error" : "warn";
        const border = { ok: "border-emerald-300 bg-emerald-50/40", error: "border-rose-300 bg-rose-50/40", warn: "border-amber-300 bg-amber-50/40" }[tone];
        const pill = { ok: "bg-emerald-600", error: "bg-rose-600", warn: "bg-amber-500" }[tone];
        return (
          <div key={c.type} className={`rounded-md border p-3 ${border}`}>
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm font-semibold">{c.type}</span>
              <span className={`rounded px-1.5 py-0.5 text-[10px] font-semibold text-white ${pill}`}>{c.status}</span>
            </div>
            {c.reason && <p className="mt-1 text-xs text-zinc-600">Reason: {c.reason}</p>}
            {c.message && (
              <p className={`mt-1 whitespace-pre-wrap break-words text-xs text-zinc-700 ${compact ? "line-clamp-6" : ""}`} title={compact ? c.message : undefined}>
                {c.message}
              </p>
            )}
            {c.lastTransitionTime && (
              <p className="mt-1 text-[11px] text-zinc-500">Última transição: {new Date(c.lastTransitionTime).toLocaleString("pt-BR")}</p>
            )}
          </div>
        );
      })}
    </div>
  );
}
