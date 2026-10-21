import type { DriftState } from "@/lib/types";

const signalLabel: Record<string, string> = {
  "argo-out-of-sync": "Argo CD out of sync with Git",
  "git-claim-mismatch": "Configuration in cluster differs from Git",
  "xr-not-synced": "Crossplane cannot reconcile composition",
  "mr-not-ready": "Cloud resource is not ready",
  "cloud-reconcile-error": "Error reconciling with cloud (possible manual drift)",
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
          <thead><tr><th className="pr-4">Field</th><th className="pr-4">Desired (Git)</th><th>Observed (cluster)</th></tr></thead>
          <tbody>
            {drift.differences.map((d) => (
              <tr key={d.field} className="font-mono">
                <td className="pr-4">{d.field}</td><td className="pr-4">{String(d.desired ?? "—")}</td><td>{String(d.observed ?? "—")}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {drift.detectedAt && <p className="mt-2 text-xs">Since {new Date(drift.detectedAt).toLocaleString("en-US")}</p>}
    </div>
  );
}
