import type { DriftState } from "@/lib/types";

const signalLabel: Record<string, string> = {
  "argo-out-of-sync": "Argo CD fora de sincronia com o Git",
  "git-claim-mismatch": "Configuração no cluster difere do Git",
  "xr-not-synced": "Crossplane não consegue reconciliar a composição",
  "mr-not-ready": "Recurso de cloud não está pronto",
  "cloud-reconcile-error": "Erro ao reconciliar com a cloud (possível alteração manual)",
};

export function DriftBanner({ drift }: { drift: DriftState }) {
  if (!drift.detected) return null;
  return (
    <div className="mt-4 rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900" role="alert">
      <p className="font-semibold">⚠ Configuration Drift</p>
      <ul className="mt-1 list-disc pl-5">
        {(drift.signals ?? []).map((s) => <li key={s}>{signalLabel[s] ?? s}</li>)}
      </ul>
      {drift.differences && drift.differences.length > 0 && (
        <table className="mt-3 text-left text-xs">
          <thead><tr><th className="pr-4">Campo</th><th className="pr-4">Desired (Git)</th><th>Observed (cluster)</th></tr></thead>
          <tbody>
            {drift.differences.map((d) => (
              <tr key={d.field} className="font-mono">
                <td className="pr-4">{d.field}</td><td className="pr-4">{String(d.desired ?? "—")}</td><td>{String(d.observed ?? "—")}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {drift.detectedAt && <p className="mt-2 text-xs">Desde {new Date(drift.detectedAt).toLocaleString("pt-BR")}</p>}
    </div>
  );
}
