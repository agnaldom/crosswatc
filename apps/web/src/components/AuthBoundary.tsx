"use client";

import { useQuery } from "@tanstack/react-query";
import { ApiError, api } from "@/lib/api";
import { RealtimeBridge } from "@/lib/realtime";
import { ErrorState, Loading } from "@/components/States";
import { Icon } from "@/components/Icon";

export function AuthBoundary({ children }: { children: React.ReactNode }) {
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, refetchInterval: 60_000 });
  const status = useQuery({ queryKey: ["auth-status"], queryFn: async () => {
    const response = await fetch("/api/v1/auth/status", { cache: "no-store" });
    if (!response.ok) throw new Error("Could not load sign-in options");
    return response.json() as Promise<{ loginEnabled: boolean }>;
  }, refetchInterval: false });
  if (me.isPending) return <div className="grid min-h-screen place-items-center"><Loading label="Checking workspace access…" /></div>;
  if (me.error instanceof ApiError && me.error.status === 401) return (
    <main className="grid min-h-screen place-items-center bg-surface px-6">
      <section className="cw-panel w-full max-w-md space-y-6 p-8">
        <span className="cw-brand-mark"><Icon name="deployed_code" className="h-5 w-5" aria-hidden /></span>
        <div><p className="cw-eyebrow mb-2">CROSSWATC WORKSPACE</p><h1 className="cw-page-title">Welcome back</h1><p className="cw-page-description">Sign in with your organization account to access your infrastructure.</p></div>
        {status.isPending ? <Loading label="Loading sign-in…" /> : status.isError ? <ErrorState error={status.error} /> : status.data.loginEnabled ? <button type="button" className="cw-button w-full justify-center" onClick={() => window.location.assign(new URL("/api/v1/auth/login", window.location.origin).href)}>Sign in with Keycloak</button> : <p className="text-sm text-on-surface-variant">Sign in through your organization’s access gateway. Browser login is not configured for this environment.</p>}
      </section>
    </main>
  );
  if (me.isError) return <div className="mx-auto mt-20 max-w-lg"><ErrorState error={me.error} /><button className="cw-button mt-4" type="button" onClick={() => me.refetch()}>Try again</button></div>;
  return <><RealtimeBridge />{children}</>;
}
