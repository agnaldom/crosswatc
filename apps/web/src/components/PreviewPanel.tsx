import { Badge } from "@/components/Badge";
import type { ChangePreview, CostDiff } from "@/lib/types";

function money(v: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: currency || "USD" }).format(v);
}

function CostLine({ cost }: { cost: CostDiff }) {
  const up = cost.monthlyDelta > 0;
  return (
    <div className="rounded-md border border-zinc-200 p-3">
      <p className="text-xs uppercase tracking-wide text-zinc-500">Estimated cost (monthly)</p>
      <p className="mt-1">
        {money(cost.current.monthly, cost.currency)} → <strong>{money(cost.new.monthly, cost.currency)}</strong>{" "}
        <span className={up ? "text-rose-700" : "text-emerald-700"}>
          ({up ? "+" : ""}{money(cost.monthlyDelta, cost.currency)}/mo)
        </span>
      </p>
      {cost.partial && <p className="mt-1 text-xs text-amber-700">Partial estimate: some components were not priced.</p>}
      <p className="mt-1 text-xs text-zinc-500">On-demand list price; does not include discounts, reservations, or taxes.</p>
    </div>
  );
}

const fmt = (v: unknown) => (v === undefined || v === null || v === "" ? "—" : String(v));

export function PreviewPanel({ preview }: { preview: ChangePreview }) {
  return (
    <div className="space-y-4 text-sm" aria-live="polite">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-medium">Policies</span>
        <Badge tone={preview.policy.result === "PASS" ? "ok" : "error"}>{preview.policy.result}</Badge>
        {preview.impact.approvalRequired && <Badge tone="warn">Approval required</Badge>}
        {preview.impact.replacementRequired && <Badge tone="error">Resource replacement</Badge>}
        {preview.impact.possibleDowntime && <Badge tone="warn">Possible downtime</Badge>}
      </div>

      {preview.changes.length > 0 && (
        <table className="w-full overflow-hidden rounded-md border border-zinc-200 text-left">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr><th className="px-3 py-2">Field</th><th className="px-3 py-2">Before</th><th className="px-3 py-2">After</th><th className="px-3 py-2">Impact</th></tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {preview.changes.map((c) => (
              <tr key={c.field}>
                <td className="px-3 py-2 font-mono text-xs">{c.field}</td>
                <td className="px-3 py-2 text-zinc-500 line-through">{fmt(c.from)}</td>
                <td className="px-3 py-2 font-medium">{fmt(c.to)}</td>
                <td className="px-3 py-2"><Badge tone={c.impact === "IN_PLACE" ? "ok" : "error"}>{c.impact}</Badge></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {preview.cost && <CostLine cost={preview.cost} />}

      {preview.policy.violations.length > 0 && (
        <ul className="space-y-1 rounded-md bg-rose-50 p-3 text-rose-800" role="alert">
          {preview.policy.violations.map((v, i) => (
            <li key={i}><span className="font-mono text-xs">{v.field ? `${v.field} · ` : ""}{v.rule}</span> — {v.message}</li>
          ))}
        </ul>
      )}
      {preview.warnings.length > 0 && (
        <ul className="space-y-1 rounded-md bg-amber-50 p-3 text-amber-800">
          {preview.warnings.map((w, i) => <li key={i}>{w}</li>)}
        </ul>
      )}
    </div>
  );
}
