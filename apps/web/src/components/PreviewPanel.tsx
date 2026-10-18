import { Badge } from "@/components/Badge";
import type { ChangePreview } from "@/lib/types";

const fmt = (v: unknown) => (v === undefined || v === null || v === "" ? "—" : String(v));

export function PreviewPanel({ preview }: { preview: ChangePreview }) {
  return (
    <div className="space-y-4 text-sm" aria-live="polite">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-medium">Policies</span>
        <Badge tone={preview.policy.result === "PASS" ? "ok" : "error"}>{preview.policy.result}</Badge>
        {preview.impact.approvalRequired && <Badge tone="warn">Requer aprovação</Badge>}
        {preview.impact.replacementRequired && <Badge tone="error">Recriação do recurso</Badge>}
        {preview.impact.possibleDowntime && <Badge tone="warn">Possível indisponibilidade</Badge>}
      </div>

      {preview.changes.length > 0 && (
        <table className="w-full overflow-hidden rounded-md border border-zinc-200 text-left">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr><th className="px-3 py-2">Campo</th><th className="px-3 py-2">Antes</th><th className="px-3 py-2">Depois</th><th className="px-3 py-2">Impacto</th></tr>
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
