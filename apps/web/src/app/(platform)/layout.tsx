"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { ErrorState, Loading } from "@/components/States";

export default function PlatformLayout({ children }: { children: React.ReactNode }) {
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, refetchInterval: false });
  if (me.isPending) return <Loading />;
  if (me.isError) return <ErrorState error={me.error} />;
  if (!me.data.sre) return <p className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">This page is restricted to the SRE/Platform team.</p>;
  return <div className="cw-workspace-page cw-catalog space-y-6">{children}</div>;
}
