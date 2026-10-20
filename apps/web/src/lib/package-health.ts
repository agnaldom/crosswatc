import { normalizeConditions } from "./conditions";
import type { CrossplanePackage } from "./types";

export interface PackageHealth {
  total: number;
  healthy: number;
  /** Packages whose conditions normalize to an error (e.g. Healthy=False). */
  unhealthy: CrossplanePackage[];
  /** Neither healthy nor failing yet (installing, unknown). */
  pending: number;
}

/** Summarizes Provider/Function health the same way the platform catalog renders it. */
export function packageHealth(items: CrossplanePackage[]): PackageHealth {
  const h: PackageHealth = { total: items.length, healthy: 0, unhealthy: [], pending: 0 };
  for (const p of items) {
    const tone = normalizeConditions(p.conditions, p.kind).tone;
    if (tone === "ok") h.healthy++;
    else if (tone === "error") h.unhealthy.push(p);
    else h.pending++;
  }
  return h;
}
