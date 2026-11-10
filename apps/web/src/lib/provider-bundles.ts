import type { CrossplanePackage } from "./types";

/** Prefer the requested publisher; preserve legacy candidates for visibility, counting each target once. */
export function bundlePackages(repositories: string[], installed: CrossplanePackage[]): CrossplanePackage[] {
  return repositories.flatMap((repo) => {
    const candidates = installed.filter((p) => packageName(p.package) === packageName(repo));
    const item = candidates.find((p) => p.package.split("@")[0].split(":")[0] === repo) ?? candidates.at(-1);
    return item ? [item] : [];
  });
}

function packageName(ref: string) {
  return ref.split("/").at(-1)?.split("@")[0].split(":")[0];
}

export function providerState(p: CrossplanePackage): { label: string; tone: "ok" | "error" | "warn"; ready: boolean; failed: boolean } {
  const conditions = p.conditions ?? [];
  const installed = conditions.find((c) => c.type === "Installed")?.status;
  const healthy = conditions.find((c) => c.type === "Healthy")?.status;
  const ready = installed === "True" && healthy === "True";
  const failed = installed === "False" || healthy === "False";
  const dependency = conditions.some((c) => /dependenc/i.test(`${c.reason ?? ""} ${c.message ?? ""}`) && c.status !== "True");
  return { label: ready ? "Ready" : failed ? "Needs attention" : dependency ? "Waiting for dependencies" : "Installing", tone: ready ? "ok" : failed ? "error" : "warn", ready, failed };
}
