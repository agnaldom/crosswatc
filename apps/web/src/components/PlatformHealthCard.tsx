"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { api } from "@/lib/api";
import { packageHealth } from "@/lib/package-health";

/** SRE-only dashboard card: Provider and Function health (GET /api/v1/platform/{providers,functions}). */
export function PlatformHealthCard() {
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, refetchInterval: false });
  if (!me.data?.sre) return null;
  return (
    <section className="mt-6 rounded-xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#0e1117]" aria-label="Platform health">
      <div className="flex items-baseline justify-between">
        <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Crossplane Platform Control Plane</h2>
        <Link href="/platform" className="text-xs text-blue-600 hover:underline dark:text-blue-400">Open catalog →</Link>
      </div>
      <div className="mt-3 grid gap-4 sm:grid-cols-2">
        <Packages kind="providers" label="Providers" />
        <Packages kind="functions" label="Functions" />
      </div>
    </section>
  );
}

function Packages({ kind, label }: { kind: "providers" | "functions"; label: string }) {
  const q = useQuery({ queryKey: ["platform", kind], queryFn: () => api.platformPackages(kind) });
  if (q.isPending) return <p className="text-sm text-zinc-500 dark:text-zinc-400">{label}: loading…</p>;
  if (q.isError) {
    return <p className="line-clamp-2 text-sm text-rose-600 dark:text-rose-400" title={q.error.message}>{label}: unavailable ({q.error.message})</p>;
  }
  const h = packageHealth(q.data.items);
  return (
    <div className="rounded-lg border border-zinc-100 bg-zinc-50/50 p-3.5 dark:border-zinc-800/80 dark:bg-zinc-900/40">
      <p className="text-sm">
        <span className="font-semibold text-zinc-900 dark:text-zinc-100">{label}</span>{" "}
        <span className="tabular-nums text-zinc-600 dark:text-zinc-400">{h.healthy}/{h.total} healthy</span>
        {h.pending > 0 && <span className="text-zinc-500 dark:text-zinc-400"> · {h.pending} pending</span>}
      </p>
      {h.unhealthy.length > 0 ? (
        <ul className="mt-2 space-y-1 text-xs">
          {h.unhealthy.map((p) => (
            <li key={p.name} className="flex items-center gap-1.5 text-rose-600 dark:text-rose-400">
              <span className="h-1.5 w-1.5 rounded-full bg-rose-500" aria-hidden />
              <span className="font-mono">{p.name}</span>
            </li>
          ))}
        </ul>
      ) : (
        h.total > 0 && <p className="mt-2 text-xs font-medium text-emerald-600 dark:text-emerald-400">All healthy and established.</p>
      )}
    </div>
  );
}
