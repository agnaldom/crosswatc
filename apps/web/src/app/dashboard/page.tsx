"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { Icon } from "@/components/Icon";
import { api, getContexts } from "@/lib/api";

export default function DashboardPage() {
  const [filterQuery, setFilterQuery] = useState("");
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const dashboardQuery = useQuery({
    queryKey: ["resources", "dashboard"],
    queryFn: api.dashboard,
    staleTime: 30_000,
  });

  const contextsQuery = useQuery({
    queryKey: ["contexts"],
    queryFn: getContexts,
    staleTime: 30_000,
  });

  const currentContext = contextsQuery.data?.currentContext || "k8s-control-plane-us-east";
  const data = dashboardQuery.data;

  const totalComposed = data?.total ?? 3;
  const readyCount = data?.health?.ready ?? 0;
  const notReadyCount = data?.health?.notReady ?? 0;
  const unknownCount = data?.health?.unknown ?? 3;

  const handleCopy = (text: string, id: string) => {
    navigator.clipboard?.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  return (
    <div className="w-full flex flex-col space-y-space-xl">
      {/* Top Action & Overview Title Bar */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-space-lg">
        <div className="space-y-space-xs">
          <div className="flex flex-wrap items-center gap-space-sm">
            <h1 className="font-headline-xl text-headline-xl text-on-surface tracking-tight font-semibold">
              Infrastructure Overview
            </h1>
            <div className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-surface-container-high text-primary font-label-sm text-label-sm shadow-sm border border-surface-container-highest/50">
              <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse"></span>
              <span>Control Plane: connected ({currentContext})</span>
              <span className="text-outline">•</span>
              <span className="text-on-surface-variant font-mono">Crossplane v1.14.0</span>
              <span className="text-outline">•</span>
              <span className="text-on-surface-variant">1 Cluster</span>
            </div>
          </div>
          <p className="font-body-md text-body-md text-on-surface-variant">
            Monitor and manage composed cloud infrastructure, provider states, and claims across your platform.
          </p>
        </div>

        {/* Action Row */}
        <div className="flex flex-wrap items-center gap-space-sm">
          <button
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded bg-surface-container hover:bg-surface-container-high transition-colors text-on-surface font-label-md text-label-md shadow-sm border border-surface-container-high/40 cursor-pointer"
            type="button"
          >
            <Icon name="schedule" className="h-4 w-4 text-on-surface-variant" />
            <span>Last 1 hour</span>
            <Icon name="expand_more" className="h-3.5 w-3.5 text-outline" />
          </button>
          <button
            onClick={() => dashboardQuery.refetch()}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded bg-surface-container hover:bg-surface-container-high transition-colors text-on-surface font-label-md text-label-md shadow-sm border border-surface-container-high/40 cursor-pointer"
            type="button"
          >
            <Icon
              name="autorenew"
              className={`h-4 w-4 text-primary ${dashboardQuery.isFetching ? "animate-spin" : ""}`}
            />
            <span>Refresh (30s)</span>
          </button>
          <Link
            href="/infrastructure"
            className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded bg-[#2563eb] text-white hover:opacity-90 font-label-md text-label-md font-medium shadow-md shadow-blue-500/20 transition-all cursor-pointer"
          >
            <Icon name="add" className="h-4 w-4 text-white" />
            <span>Request Resource</span>
          </Link>
        </div>
      </div>

      {/* SECTION 1: INFRASTRUCTURE KPIS (6 Cards) */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-space-md">
        {/* KPI 1: Total Composed */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between hover:bg-surface-container transition-colors">
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Total Composed
            </span>
            <div className="w-7 h-7 rounded bg-surface-container-high flex items-center justify-center text-primary">
              <Icon name="layers" className="h-4 w-4 text-primary" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold">{totalComposed}</span>
              <span className="font-label-sm text-label-sm px-1.5 py-0.5 rounded bg-surface-container-highest text-primary font-mono">
                XRD bound
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">Across all namespaces</p>
          </div>
        </div>

        {/* KPI 2: Ready Resources */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between hover:bg-surface-container transition-colors">
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Ready Resources
            </span>
            <div className="w-7 h-7 rounded bg-surface-container-high flex items-center justify-center text-secondary">
              <Icon name="check_circle" className="h-4 w-4 text-secondary" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold">{readyCount}</span>
              <span className="inline-flex items-center gap-1 font-label-sm text-label-sm text-secondary-fixed">
                <span className="w-1.5 h-1.5 rounded-full bg-secondary"></span>
                {totalComposed > 0 ? Math.round((readyCount / totalComposed) * 100) : 0}% Ready
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">
              {readyCount} of {totalComposed} reconciled ready
            </p>
          </div>
        </div>

        {/* KPI 3: Not Ready */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between hover:bg-surface-container transition-colors">
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Not Ready
            </span>
            <div className="w-7 h-7 rounded bg-surface-container-high flex items-center justify-center text-error">
              <Icon name="cancel" className="h-4 w-4 text-error" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold">{notReadyCount}</span>
              <span className="inline-flex items-center gap-1 font-label-sm text-label-sm text-error">
                <span className="w-1.5 h-1.5 rounded-full bg-error"></span>
                {notReadyCount} Failed
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">No hard reconciliation errors</p>
          </div>
        </div>

        {/* KPI 4: Unknown Health */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between hover:bg-surface-container transition-colors">
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Unknown Health
            </span>
            <div className="w-7 h-7 rounded bg-surface-container-high flex items-center justify-center text-secondary-fixed">
              <Icon name="help" className="h-4 w-4 text-secondary-fixed" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold">{unknownCount}</span>
              <span className="inline-flex items-center gap-1 font-label-sm text-label-sm text-secondary px-1.5 py-0.5 rounded bg-surface-container-highest">
                Pending Probes
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1 truncate" title="Pending health probe / missing visibility">
              Awaiting probe / XRD read
            </p>
          </div>
        </div>

        {/* KPI 5: Healthy Providers */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between hover:bg-surface-container transition-colors">
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Healthy Providers
            </span>
            <div className="w-7 h-7 rounded bg-surface-container-high flex items-center justify-center text-primary">
              <Icon name="hub" className="h-4 w-4 text-primary" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold">
                11<span className="text-on-surface-variant text-headline-md font-normal">/13</span>
              </span>
              <span className="font-label-sm text-label-sm text-primary font-mono">84.6%</span>
            </div>
            <div className="w-full bg-surface-container-highest rounded-full h-1.5 mt-2 overflow-hidden">
              <div className="bg-[#2563eb] h-full rounded-full" style={{ width: "84.6%" }}></div>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1.5">2 degraded (Compute, Net)</p>
          </div>
        </div>

        {/* KPI 6: Pending Approvals */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between hover:bg-surface-container transition-colors">
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Pending Approvals
            </span>
            <div className="relative w-7 h-7 rounded bg-surface-container-high flex items-center justify-center text-secondary-fixed">
              <Icon name="approval" className="h-4 w-4 text-secondary-fixed" />
              <span className="absolute top-0 right-0 w-2 h-2 rounded-full bg-primary animate-ping"></span>
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold">1</span>
              <span className="font-label-sm text-label-sm text-on-surface-variant">Requires review</span>
            </div>
            <Link
              className="inline-flex items-center gap-1 font-label-sm text-label-sm text-primary hover:underline mt-1 font-medium"
              href="/infrastructure"
            >
              <span>Review requests</span>
              <Icon name="arrow_forward" className="h-3.5 w-3.5 text-primary" />
            </Link>
          </div>
        </div>
      </div>

      {/* SECTION 2: RESOURCE HEALTH & DISTRIBUTION (3 Cards) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-space-md">
        {/* Card A: Resource Health Donut */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between">
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Resource Health</h2>
              <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant">
                {totalComposed} Monitored
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Composite resource reconciliation readiness
            </p>

            {/* Donut Graphic */}
            <div className="relative flex items-center justify-center my-space-lg">
              <svg className="w-36 h-36 transform -rotate-90" viewBox="0 0 100 100">
                <circle
                  className="text-surface-container-highest"
                  cx="50"
                  cy="50"
                  fill="none"
                  r="40"
                  stroke="currentColor"
                  strokeWidth="12"
                />
                <circle
                  className="text-[#2563eb]"
                  cx="50"
                  cy="50"
                  fill="none"
                  r="40"
                  stroke="currentColor"
                  strokeDasharray="251.2"
                  strokeDashoffset="0"
                  strokeLinecap="round"
                  strokeWidth="12"
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                <span className="font-display-lg text-display-lg text-on-surface font-bold leading-none">
                  {totalComposed}
                </span>
                <span className="font-label-sm text-label-sm text-on-surface-variant font-mono mt-1">Total XR</span>
              </div>
            </div>

            {/* Legend */}
            <div className="space-y-2 mt-2">
              <div className="flex items-center justify-between text-body-sm font-body-sm">
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-secondary"></span>
                  <span className="text-on-surface">Ready</span>
                </div>
                <span className="font-mono text-on-surface-variant">
                  {readyCount} ({totalComposed > 0 ? Math.round((readyCount / totalComposed) * 100) : 0}%)
                </span>
              </div>
              <div className="flex items-center justify-between text-body-sm font-body-sm">
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-error"></span>
                  <span className="text-on-surface">Not Ready / Failed</span>
                </div>
                <span className="font-mono text-on-surface-variant">
                  {notReadyCount} ({totalComposed > 0 ? Math.round((notReadyCount / totalComposed) * 100) : 0}%)
                </span>
              </div>
              <div className="flex items-center justify-between text-body-sm font-body-sm">
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-[#2563eb]"></span>
                  <span className="text-on-surface">Unknown / Syncing</span>
                </div>
                <span className="font-mono text-primary font-medium">
                  {unknownCount} ({totalComposed > 0 ? Math.round((unknownCount / totalComposed) * 100) : 100}%)
                </span>
              </div>
            </div>
          </div>

          <div className="mt-space-md p-space-sm rounded bg-surface-container font-body-sm text-body-sm text-on-surface-variant flex items-start gap-2 border border-surface-container-high/30">
            <Icon name="info" className="h-4 w-4 text-primary shrink-0 mt-0.5" />
            <span>Health visibility probes awaiting XRD readiness status evaluation.</span>
          </div>
        </div>

        {/* Card B: Cloud Distribution */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between">
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Cloud Distribution</h2>
              <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-high text-primary font-mono">
                1 Target Provider
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Resources managed across registered hyperscalers
            </p>

            <div className="space-y-space-md mt-space-lg">
              {/* Azure Row */}
              <div className="space-y-1.5">
                <div className="flex items-center justify-between font-body-sm text-body-sm">
                  <div className="flex items-center gap-2">
                    <Icon name="cloud" className="h-4 w-4 text-primary" />
                    <span className="font-medium text-on-surface">Microsoft Azure</span>
                  </div>
                  <span className="font-mono text-primary font-medium">3 (100%)</span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden">
                  <div className="bg-[#2563eb] h-full rounded-full transition-all duration-500" style={{ width: "100%" }}></div>
                </div>
                <span className="font-label-sm text-label-sm text-outline font-mono">sub: prod-hybrid-mesh-01</span>
              </div>

              {/* AWS Row */}
              <div className="space-y-1.5 opacity-60">
                <div className="flex items-center justify-between font-body-sm text-body-sm">
                  <div className="flex items-center gap-2">
                    <Icon name="cloud_queue" className="h-4 w-4 text-on-surface-variant" />
                    <span className="text-on-surface-variant">Amazon Web Services</span>
                  </div>
                  <span className="font-mono text-on-surface-variant">0 (0%)</span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden">
                  <div className="bg-outline h-full rounded-full" style={{ width: "0%" }}></div>
                </div>
                <span className="font-label-sm text-label-sm text-outline">Configured, 0 active claims</span>
              </div>

              {/* Huawei Cloud Row */}
              <div className="space-y-1.5 opacity-60">
                <div className="flex items-center justify-between font-body-sm text-body-sm">
                  <div className="flex items-center gap-2">
                    <Icon name="cloud_queue" className="h-4 w-4 text-on-surface-variant" />
                    <span className="text-on-surface-variant">Huawei Cloud</span>
                  </div>
                  <span className="font-mono text-on-surface-variant">0 (0%)</span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden">
                  <div className="bg-outline h-full rounded-full" style={{ width: "0%" }}></div>
                </div>
                <span className="font-label-sm text-label-sm text-outline">Configured, 0 active claims</span>
              </div>
            </div>
          </div>

          <div className="mt-space-md p-space-sm rounded bg-surface-container font-body-sm text-body-sm text-on-surface-variant flex items-center gap-2 border border-surface-container-high/30">
            <Icon name="check" className="h-4 w-4 text-on-surface-variant" />
            <span>Current active footprint concentrated in Azure East US 2.</span>
          </div>
        </div>

        {/* Card C: Environment Distribution */}
        <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between">
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Environment Distribution</h2>
              <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-high text-primary font-mono">
                1 active
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Logical partition topology for current deployment
            </p>

            <div className="space-y-space-md mt-space-lg">
              {/* Dev */}
              <div className="p-space-md rounded bg-surface-container space-y-2 border border-surface-container-high/30">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2.5 h-2.5 rounded-sm bg-[#2563eb]"></span>
                    <span className="font-label-md text-label-md font-semibold text-on-surface">Development (dev)</span>
                  </div>
                  <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-[#2563eb] text-white font-mono">
                    {totalComposed} resources
                  </span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden">
                  <div className="bg-[#2563eb] h-full rounded-full" style={{ width: "100%" }}></div>
                </div>
                <p className="font-label-sm text-label-sm text-on-surface-variant">
                  Primary active staging lane for squad platform tests
                </p>
              </div>

              {/* Staging */}
              <div className="p-space-md rounded bg-surface-container-low opacity-60 space-y-1.5 border border-surface-container-high/20">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2.5 h-2.5 rounded-sm bg-outline"></span>
                    <span className="font-label-md text-label-md font-medium text-on-surface-variant">Staging (qa)</span>
                  </div>
                  <span className="font-mono text-label-sm text-label-sm text-outline">0 active</span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-1.5 border border-dashed border-outline-variant"></div>
              </div>

              {/* Production */}
              <div className="p-space-md rounded bg-surface-container-low opacity-60 space-y-1.5 border border-surface-container-high/20">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2.5 h-2.5 rounded-sm bg-outline"></span>
                    <span className="font-label-md text-label-md font-medium text-on-surface-variant">Production (prod)</span>
                  </div>
                  <span className="font-mono text-label-sm text-label-sm text-outline">0 active</span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-1.5 border border-dashed border-outline-variant"></div>
              </div>
            </div>
          </div>

          <div className="mt-space-md flex items-center justify-between font-label-sm text-label-sm text-on-surface-variant">
            <span>Namespace isolation enabled</span>
            <span className="font-mono text-primary">Role: platform-admin</span>
          </div>
        </div>
      </div>

      {/* SECTION 3: CROSSPLANE PLATFORM HEALTH */}
      <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container space-y-space-md">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-space-sm">
          <div>
            <div className="flex items-center gap-2">
              <Icon name="memory" className="h-5 w-5 text-primary" />
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">
                Crossplane Control Plane
              </h2>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Control plane reconciliation loop status and core package controllers
            </p>
          </div>
          <Link
            className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:underline font-medium"
            href="/platform"
          >
            <span>View platform details</span>
            <Icon name="arrow_forward" className="h-4 w-4 text-primary" />
          </Link>
        </div>

        {/* Metric Pills Row */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-space-sm">
          <div className="p-space-sm rounded bg-surface-container flex items-center justify-between border border-surface-container-high/30">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-secondary-fixed"></span>
              <span className="font-body-sm text-body-sm text-on-surface">Providers</span>
            </div>
            <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-high text-secondary-fixed font-mono">
              11 / 13 healthy
            </span>
          </div>
          <div className="p-space-sm rounded bg-surface-container flex items-center justify-between border border-surface-container-high/30">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-outline"></span>
              <span className="font-body-sm text-body-sm text-on-surface">Functions</span>
            </div>
            <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-high text-outline font-mono">
              0 installed
            </span>
          </div>
          <div className="p-space-sm rounded bg-surface-container flex items-center justify-between border border-surface-container-high/30">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-secondary-fixed"></span>
              <span className="font-body-sm text-body-sm text-on-surface">Compositions</span>
            </div>
            <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant font-mono">
              Unavailable (RBAC)
            </span>
          </div>
          <div className="p-space-sm rounded bg-surface-container flex items-center justify-between border border-surface-container-high/30">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-outline"></span>
              <span className="font-body-sm text-body-sm text-on-surface">XRDs</span>
            </div>
            <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant font-mono">
              Awaiting sync
            </span>
          </div>
        </div>

        {/* Health Progress Bar */}
        <div className="space-y-1">
          <div className="flex items-center justify-between font-label-sm text-label-sm">
            <span className="text-on-surface-variant">Overall Provider Reconciliation Health</span>
            <span className="font-mono text-on-surface font-semibold">84.6% Operational</span>
          </div>
          <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden flex">
            <div className="bg-[#2563eb] h-full" style={{ width: "84.6%" }}></div>
            <div className="bg-error h-full" style={{ width: "15.4%" }}></div>
          </div>
        </div>

        {/* Operational Issues List */}
        <div className="overflow-x-auto">
          <table className="w-full text-left font-body-sm text-body-sm">
            <thead>
              <tr className="text-outline uppercase font-mono text-[11px] tracking-wider bg-surface-container">
                <th className="py-2.5 px-space-md rounded-l">Provider Package</th>
                <th className="py-2.5 px-space-md">Version</th>
                <th className="py-2.5 px-space-md">Reconciliation Condition</th>
                <th className="py-2.5 px-space-md">Error Diagnostic</th>
                <th className="py-2.5 px-space-md text-right rounded-r">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-container/40">
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-space-sm px-space-md">
                  <div className="flex items-center gap-2 font-mono font-medium text-on-surface">
                    <Icon name="deployed_code" className="h-4 w-4 text-error shrink-0" />
                    <span>provider-azure-compute</span>
                  </div>
                </td>
                <td className="py-space-sm px-space-md font-mono text-on-surface-variant">v0.42.1</td>
                <td className="py-space-sm px-space-md">
                  <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-error-container text-on-error-container font-label-sm text-label-sm font-semibold">
                    <span className="w-1.5 h-1.5 rounded-full bg-error animate-pulse"></span>
                    Unhealthy / Degraded
                  </span>
                </td>
                <td className="py-space-sm px-space-md text-error font-mono text-code-sm">
                  Reconciliation backoff: Provider revision Unhealthy. Connection reset by peer.
                </td>
                <td className="py-space-sm px-space-md text-right">
                  <Link
                    href="/platform?tab=Providers"
                    className="px-2.5 py-1 rounded bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-sm text-label-sm transition-colors inline-block"
                  >
                    Inspect Details
                  </Link>
                </td>
              </tr>
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-space-sm px-space-md">
                  <div className="flex items-center gap-2 font-mono font-medium text-on-surface">
                    <Icon name="deployed_code" className="h-4 w-4 text-error shrink-0" />
                    <span>provider-azure-network</span>
                  </div>
                </td>
                <td className="py-space-sm px-space-md font-mono text-on-surface-variant">v0.41.0</td>
                <td className="py-space-sm px-space-md">
                  <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-error-container text-on-error-container font-label-sm text-label-sm font-semibold">
                    <span className="w-1.5 h-1.5 rounded-full bg-error"></span>
                    Unhealthy
                  </span>
                </td>
                <td className="py-space-sm px-space-md text-error font-mono text-code-sm">
                  CRD install timeout: NetworkSecurityGroup validation failed during schema apply.
                </td>
                <td className="py-space-sm px-space-md text-right">
                  <Link
                    href="/platform?tab=Providers"
                    className="px-2.5 py-1 rounded bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-sm text-label-sm transition-colors inline-block"
                  >
                    Inspect Details
                  </Link>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* SECTION 4: OPERATIONAL ATTENTION & PROVISIONING WORKFLOW (Two-Column) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-space-md">
        {/* Left Column: Needs Attention (7 cols) */}
        <div className="lg:col-span-7 p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between space-y-space-md">
          <div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Icon name="notification_important" className="h-5 w-5 text-error" />
                <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Needs Attention</h2>
              </div>
              <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-error-container text-on-error-container font-mono font-semibold">
                4 items
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Priority alerts and configuration anomalies impacting reconciliation
            </p>

            <div className="space-y-space-sm mt-space-md">
              {/* Item 1: Critical */}
              <div className="p-space-sm rounded bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2 hover:bg-surface-container-high transition-colors border border-surface-container-high/30">
                <div className="flex items-start gap-2.5 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-1.5 py-0.5 rounded bg-error-container text-on-error-container uppercase font-mono font-semibold">
                    Critical
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-medium text-on-surface truncate">
                      provider-azure-compute
                    </p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      Provider pod reconciliation failing in backoff loop
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center flex-shrink-0">
                  <Link href="/platform?tab=Providers" className="font-label-sm text-label-sm text-primary hover:underline">
                    Resolve
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link href="/platform?tab=Providers" className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface">
                    View
                  </Link>
                </div>
              </div>

              {/* Item 2: Warning */}
              <div className="p-space-sm rounded bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2 hover:bg-surface-container-high transition-colors border border-surface-container-high/30">
                <div className="flex items-start gap-2.5 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-1.5 py-0.5 rounded bg-secondary-container text-secondary-fixed uppercase font-mono font-semibold">
                    Warning
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-medium text-on-surface truncate">vm-payments-batch</p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      Missing resource health probe configuration from XRD definition
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center flex-shrink-0">
                  <Link href="/infrastructure" className="font-label-sm text-label-sm text-primary hover:underline">
                    Resolve
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link href="/infrastructure" className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface">
                    View
                  </Link>
                </div>
              </div>

              {/* Item 3: Warning */}
              <div className="p-space-sm rounded bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2 hover:bg-surface-container-high transition-colors border border-surface-container-high/30">
                <div className="flex items-start gap-2.5 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-1.5 py-0.5 rounded bg-secondary-container text-secondary-fixed uppercase font-mono font-semibold">
                    Warning
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-medium text-on-surface truncate">storage-cards</p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      Claim not found in target cluster namespace: storage-system
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center flex-shrink-0">
                  <Link href="/infrastructure" className="font-label-sm text-label-sm text-primary hover:underline">
                    Resolve
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link href="/infrastructure" className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface">
                    View
                  </Link>
                </div>
              </div>

              {/* Item 4: Info */}
              <div className="p-space-sm rounded bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2 hover:bg-surface-container-high transition-colors border border-surface-container-high/30">
                <div className="flex items-start gap-2.5 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-1.5 py-0.5 rounded bg-surface-container-highest text-primary font-mono uppercase font-semibold">
                    Info
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-medium text-on-surface truncate">pgsql-payments-dev</p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      PostgreSQL Server binding established, awaiting consumer secret export
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center flex-shrink-0">
                  <Link href="/infrastructure" className="font-label-sm text-label-sm text-primary hover:underline">
                    Export
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link href="/infrastructure" className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface">
                    View
                  </Link>
                </div>
              </div>
            </div>
          </div>

          <div className="pt-2 flex items-center justify-between font-label-sm text-label-sm text-on-surface-variant">
            <span>Active filter: Severity ≥ Warning</span>
            <Link className="text-primary hover:underline" href="/infrastructure">
              Configure alert routing →
            </Link>
          </div>
        </div>

        {/* Right Column: Active Provisioning Requests (5 cols) */}
        <div className="lg:col-span-5 p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container flex flex-col justify-between space-y-space-md">
          <div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Icon name="published_with_changes" className="h-5 w-5 text-primary" />
                <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">
                  Active Provisioning Requests
                </h2>
              </div>
              <span className="font-label-sm text-label-sm px-2 py-0.5 rounded-full bg-secondary-container text-secondary-fixed font-mono font-medium">
                1 Active
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Pending approval gates and progressive rollout stages
            </p>

            {/* Request Card */}
            <div className="mt-space-md p-space-md rounded-lg bg-surface-container space-y-space-md border border-surface-container-high/40">
              <div className="flex items-start justify-between">
                <div>
                  <span className="font-headline-md text-headline-md text-on-surface font-semibold font-mono">
                    vm-payments-batch
                  </span>
                  <div className="flex items-center gap-2 mt-0.5 font-label-sm text-label-sm text-on-surface-variant">
                    <span>
                      Type: <strong className="text-on-surface font-mono">VirtualMachine</strong>
                    </span>
                    <span>•</span>
                    <span>
                      Env: <strong className="text-on-surface">Dev</strong>
                    </span>
                  </div>
                </div>
                <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-secondary-container text-secondary-fixed font-label-sm text-label-sm font-semibold">
                  <span className="w-1.5 h-1.5 rounded-full bg-primary animate-ping"></span>
                  WAITING_APPROVAL
                </span>
              </div>

              <div className="font-body-sm text-body-sm text-on-surface-variant flex items-center gap-4">
                <span>
                  Op: <strong className="text-on-surface">CREATE</strong>
                </span>
                <span>
                  Rev: <strong className="text-on-surface font-mono">#1</strong>
                </span>
                <span>
                  By: <strong className="text-on-surface">dev-squad-core</strong>
                </span>
              </div>

              {/* Lifecycle Pipeline Stepper */}
              <div className="space-y-1.5 pt-2">
                <span className="font-label-sm text-label-sm text-outline uppercase font-mono tracking-wider">
                  Lifecycle Pipeline
                </span>
                <div className="grid grid-cols-5 gap-1 text-center font-label-sm text-label-sm">
                  {/* Step 1: Requested */}
                  <div className="space-y-1">
                    <div className="h-1.5 rounded-full bg-[#2563eb]"></div>
                    <span className="text-on-surface font-medium truncate block text-[10px]">Req</span>
                  </div>
                  {/* Step 2: Validating */}
                  <div className="space-y-1">
                    <div className="h-1.5 rounded-full bg-[#2563eb]"></div>
                    <span className="text-on-surface font-medium truncate block text-[10px]">Valid</span>
                  </div>
                  {/* Step 3: Approval */}
                  <div className="space-y-1">
                    <div className="h-1.5 rounded-full bg-secondary-fixed animate-pulse"></div>
                    <span className="text-secondary-fixed font-bold truncate block text-[10px]">Approve</span>
                  </div>
                  {/* Step 4: Provisioning */}
                  <div className="space-y-1">
                    <div className="h-1.5 rounded-full bg-surface-container-highest"></div>
                    <span className="text-outline truncate block text-[10px]">Prov</span>
                  </div>
                  {/* Step 5: Ready */}
                  <div className="space-y-1">
                    <div className="h-1.5 rounded-full bg-surface-container-highest"></div>
                    <span className="text-outline truncate block text-[10px]">Ready</span>
                  </div>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="pt-2 flex flex-wrap items-center gap-2">
                <button
                  className="flex-1 py-1.5 px-3 rounded bg-[#2563eb] text-white hover:opacity-90 font-label-md text-label-md font-medium transition-colors text-center shadow-sm cursor-pointer"
                  type="button"
                >
                  Approve Request
                </button>
                <button
                  className="py-1.5 px-3 rounded bg-surface-container-high hover:bg-surface-container-highest text-error font-label-md text-label-md transition-colors cursor-pointer"
                  type="button"
                >
                  Reject
                </button>
                <Link
                  href="/infrastructure"
                  className="py-1.5 px-2.5 rounded bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-md text-label-md transition-colors inline-flex items-center justify-center"
                >
                  <Icon name="visibility" className="h-4 w-4" />
                </Link>
              </div>
            </div>
          </div>

          <div className="pt-2 flex items-center justify-between font-label-sm text-label-sm text-on-surface-variant">
            <span>Requires 1 SRE quorum approval</span>
            <span className="font-mono text-outline">Timeout: 23h 12m</span>
          </div>
        </div>
      </div>

      {/* SECTION 5: RECENT INFRASTRUCTURE ACTIVITY (Full Table) */}
      <div className="p-space-lg rounded-xl bg-surface-container-low shadow-sm border border-surface-container space-y-space-md">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-space-md">
          <div>
            <div className="flex items-center gap-2">
              <Icon name="history" className="h-5 w-5 text-primary" />
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">
                Recent Events &amp; Activity
              </h2>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Real-time audit log of Crossplane composition reconciliation iterations
            </p>
          </div>
          <Link
            className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:underline font-medium"
            href="/infrastructure"
          >
            <span>View all 128 events</span>
            <Icon name="arrow_forward" className="h-4 w-4 text-primary" />
          </Link>
        </div>

        {/* Controls & Filter Toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-space-sm pt-1">
          <div className="flex flex-wrap items-center gap-2 flex-1 max-w-xl">
            <div className="relative flex-1 min-w-[200px]">
              <div className="absolute left-2.5 top-1/2 -translate-y-1/2 text-outline pointer-events-none">
                <Icon name="search" className="h-4 w-4 text-outline" />
              </div>
              <input
                value={filterQuery}
                onChange={(e) => setFilterQuery(e.target.value)}
                className="w-full pl-8 pr-3 py-1.5 rounded bg-surface-container text-on-surface placeholder:text-outline font-body-sm text-body-sm focus:outline-none focus:ring-1 focus:ring-primary shadow-inner border border-surface-container-high/40"
                placeholder="Filter events by name, message..."
                type="text"
              />
            </div>
            <div className="relative">
              <button
                className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-surface-container hover:bg-surface-container-high text-on-surface font-label-sm text-label-sm transition-colors border border-surface-container-high/40"
                type="button"
              >
                <span>Severity: All</span>
                <Icon name="expand_more" className="h-3.5 w-3.5 text-outline" />
              </button>
            </div>
            <div className="relative">
              <button
                className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-surface-container hover:bg-surface-container-high text-on-surface font-label-sm text-label-sm transition-colors border border-surface-container-high/40"
                type="button"
              >
                <span>Type: All Resources</span>
                <Icon name="expand_more" className="h-3.5 w-3.5 text-outline" />
              </button>
            </div>
          </div>
          <div className="flex items-center gap-1 text-outline font-label-sm text-label-sm">
            <span>Showing latest 6 items</span>
          </div>
        </div>

        {/* Dense Table */}
        <div className="overflow-x-auto">
          <table className="w-full text-left font-body-sm text-body-sm">
            <thead>
              <tr className="text-outline uppercase font-mono text-[11px] tracking-wider bg-surface-container">
                <th className="py-2.5 px-space-md rounded-l">Severity</th>
                <th className="py-2.5 px-space-md">Time</th>
                <th className="py-2.5 px-space-md">Resource</th>
                <th className="py-2.5 px-space-md">Namespace</th>
                <th className="py-2.5 px-space-md">Event Message</th>
                <th className="py-2.5 px-space-md">Status</th>
                <th className="py-2.5 px-space-md text-right rounded-r">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-container/30 font-code-sm">
              {/* Event Row 1 */}
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-secondary-container text-secondary-fixed font-label-sm text-label-sm font-medium">
                    <span className="w-1.5 h-1.5 rounded-full bg-secondary-fixed"></span>
                    Warning
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-on-surface-variant whitespace-nowrap font-mono text-[11px]">
                  2m ago
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface font-medium whitespace-nowrap">
                  <div className="flex items-center gap-1.5">
                    <Icon name="data_object" className="h-4 w-4 text-primary shrink-0" />
                    <span>vm-payments-batch</span>
                  </div>
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface-variant">default</td>
                <td className="py-2.5 px-space-md text-on-surface truncate max-w-md">
                  Claim not found in cluster namespace
                </td>
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-highest text-secondary-fixed">
                    Unhealthy
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-right whitespace-nowrap">
                  <Link
                    href="/infrastructure"
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors inline-block"
                    title="Inspect"
                  >
                    <Icon name="open_in_new" className="h-4 w-4" />
                  </Link>
                  <button
                    onClick={() => handleCopy("vm-payments-batch", "ev1")}
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors cursor-pointer"
                    title={copiedId === "ev1" ? "Copied!" : "Copy Manifest"}
                    type="button"
                  >
                    <Icon
                      name={copiedId === "ev1" ? "check" : "content_copy"}
                      className={`h-4 w-4 ${copiedId === "ev1" ? "text-emerald-500" : ""}`}
                    />
                  </button>
                </td>
              </tr>

              {/* Event Row 2 */}
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-secondary-container text-secondary-fixed font-label-sm text-label-sm font-medium">
                    <span className="w-1.5 h-1.5 rounded-full bg-secondary-fixed"></span>
                    Warning
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-on-surface-variant whitespace-nowrap font-mono text-[11px]">
                  14m ago
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface font-medium whitespace-nowrap">
                  <div className="flex items-center gap-1.5">
                    <Icon name="inventory_2" className="h-4 w-4 text-primary shrink-0" />
                    <span>storage-cards</span>
                  </div>
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface-variant">storage-system</td>
                <td className="py-2.5 px-space-md text-on-surface truncate max-w-md">
                  Claim not found in cluster namespace
                </td>
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-highest text-secondary-fixed">
                    Unhealthy
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-right whitespace-nowrap">
                  <Link
                    href="/infrastructure"
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors inline-block"
                    title="Inspect"
                  >
                    <Icon name="open_in_new" className="h-4 w-4" />
                  </Link>
                  <button
                    onClick={() => handleCopy("storage-cards", "ev2")}
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors cursor-pointer"
                    title={copiedId === "ev2" ? "Copied!" : "Copy Manifest"}
                    type="button"
                  >
                    <Icon
                      name={copiedId === "ev2" ? "check" : "content_copy"}
                      className={`h-4 w-4 ${copiedId === "ev2" ? "text-emerald-500" : ""}`}
                    />
                  </button>
                </td>
              </tr>

              {/* Event Row 3 */}
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-secondary-container text-secondary-fixed font-label-sm text-label-sm font-medium">
                    <span className="w-1.5 h-1.5 rounded-full bg-secondary-fixed"></span>
                    Warning
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-on-surface-variant whitespace-nowrap font-mono text-[11px]">
                  42m ago
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface font-medium whitespace-nowrap">
                  <div className="flex items-center gap-1.5">
                    <Icon name="database" className="h-4 w-4 text-primary shrink-0" />
                    <span>pgsql-payments-dev</span>
                  </div>
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface-variant">core-payments</td>
                <td className="py-2.5 px-space-md text-on-surface truncate max-w-md">
                  Claim not found in cluster namespace
                </td>
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-highest text-secondary-fixed">
                    Unhealthy
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-right whitespace-nowrap">
                  <Link
                    href="/infrastructure"
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors inline-block"
                    title="Inspect"
                  >
                    <Icon name="open_in_new" className="h-4 w-4" />
                  </Link>
                  <button
                    onClick={() => handleCopy("pgsql-payments-dev", "ev3")}
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors cursor-pointer"
                    title={copiedId === "ev3" ? "Copied!" : "Copy Manifest"}
                    type="button"
                  >
                    <Icon
                      name={copiedId === "ev3" ? "check" : "content_copy"}
                      className={`h-4 w-4 ${copiedId === "ev3" ? "text-emerald-500" : ""}`}
                    />
                  </button>
                </td>
              </tr>

              {/* Event Row 4 */}
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-surface-container-highest text-secondary font-label-sm text-label-sm font-medium">
                    <span className="w-1.5 h-1.5 rounded-full bg-secondary"></span>
                    Normal
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-on-surface-variant whitespace-nowrap font-mono text-[11px]">
                  1h ago
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface font-medium whitespace-nowrap">
                  <div className="flex items-center gap-1.5">
                    <Icon name="database" className="h-4 w-4 text-primary shrink-0" />
                    <span>pgsql-payments-dev</span>
                  </div>
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface-variant">core-payments</td>
                <td className="py-2.5 px-space-md text-on-surface truncate max-w-md">
                  Revision 1: SUCCEEDED — Composite resource reconciled
                </td>
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-highest text-secondary">
                    Succeeded
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-right whitespace-nowrap">
                  <Link
                    href="/infrastructure"
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors inline-block"
                    title="Inspect"
                  >
                    <Icon name="open_in_new" className="h-4 w-4" />
                  </Link>
                  <button
                    onClick={() => handleCopy("pgsql-payments-dev", "ev4")}
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors cursor-pointer"
                    title={copiedId === "ev4" ? "Copied!" : "Copy Manifest"}
                    type="button"
                  >
                    <Icon
                      name={copiedId === "ev4" ? "check" : "content_copy"}
                      className={`h-4 w-4 ${copiedId === "ev4" ? "text-emerald-500" : ""}`}
                    />
                  </button>
                </td>
              </tr>

              {/* Event Row 5 */}
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-surface-container-highest text-primary font-label-sm text-label-sm font-medium">
                    <span className="w-1.5 h-1.5 rounded-full bg-[#2563eb] animate-pulse"></span>
                    Reconciling
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-on-surface-variant whitespace-nowrap font-mono text-[11px]">
                  2h ago
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface font-medium whitespace-nowrap">
                  <div className="flex items-center gap-1.5">
                    <Icon name="data_object" className="h-4 w-4 text-primary shrink-0" />
                    <span>vm-payments-batch</span>
                  </div>
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface-variant">default</td>
                <td className="py-2.5 px-space-md text-on-surface truncate max-w-md">
                  Revision 1: RECONCILING — Applying Terraform / Azure ARM payload
                </td>
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-highest text-primary">
                    Reconciling
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-right whitespace-nowrap">
                  <Link
                    href="/infrastructure"
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors inline-block"
                    title="Inspect"
                  >
                    <Icon name="open_in_new" className="h-4 w-4" />
                  </Link>
                  <button
                    onClick={() => handleCopy("vm-payments-batch", "ev5")}
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors cursor-pointer"
                    title={copiedId === "ev5" ? "Copied!" : "Copy Manifest"}
                    type="button"
                  >
                    <Icon
                      name={copiedId === "ev5" ? "check" : "content_copy"}
                      className={`h-4 w-4 ${copiedId === "ev5" ? "text-emerald-500" : ""}`}
                    />
                  </button>
                </td>
              </tr>

              {/* Event Row 6 */}
              <tr className="hover:bg-surface-container transition-colors">
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-surface-container-highest text-secondary font-label-sm text-label-sm font-medium">
                    <span className="w-1.5 h-1.5 rounded-full bg-secondary"></span>
                    Normal
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-on-surface-variant whitespace-nowrap font-mono text-[11px]">
                  3h ago
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface font-medium whitespace-nowrap">
                  <div className="flex items-center gap-1.5">
                    <Icon name="inventory_2" className="h-4 w-4 text-primary shrink-0" />
                    <span>storage-cards</span>
                  </div>
                </td>
                <td className="py-2.5 px-space-md font-mono text-on-surface-variant">storage-system</td>
                <td className="py-2.5 px-space-md text-on-surface truncate max-w-md">
                  Revision 1: MERGED — Pull request #142 merged by platform-bot
                </td>
                <td className="py-2.5 px-space-md whitespace-nowrap">
                  <span className="font-label-sm text-label-sm px-2 py-0.5 rounded bg-surface-container-highest text-secondary">
                    Normal
                  </span>
                </td>
                <td className="py-2.5 px-space-md text-right whitespace-nowrap">
                  <Link
                    href="/infrastructure"
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors inline-block"
                    title="Inspect"
                  >
                    <Icon name="open_in_new" className="h-4 w-4" />
                  </Link>
                  <button
                    onClick={() => handleCopy("storage-cards", "ev6")}
                    className="p-1 rounded hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors cursor-pointer"
                    title={copiedId === "ev6" ? "Copied!" : "Copy Manifest"}
                    type="button"
                  >
                    <Icon
                      name={copiedId === "ev6" ? "check" : "content_copy"}
                      className={`h-4 w-4 ${copiedId === "ev6" ? "text-emerald-500" : ""}`}
                    />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        {/* Footer Pagination */}
        <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-space-sm font-label-sm text-label-sm text-on-surface-variant">
          <span>Showing 6 of 128 events • Page 1 of 22</span>
          <div className="flex items-center gap-1">
            <button
              className="px-2.5 py-1 rounded bg-surface-container text-outline opacity-50 cursor-not-allowed border border-surface-container-high/30"
              disabled
              type="button"
            >
              Previous
            </button>
            <button
              className="px-2.5 py-1 rounded bg-surface-container hover:bg-surface-container-high text-on-surface transition-colors border border-surface-container-high/30 cursor-pointer"
              type="button"
            >
              Next
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
