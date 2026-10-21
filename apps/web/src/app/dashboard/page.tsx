"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useMemo, useState } from "react";
import { Icon } from "@/components/Icon";
import { api, getContexts } from "@/lib/api";

type TimeRange = "15m" | "1h" | "24h" | "7d";
type SeverityFilter = "ALL" | "WARNING" | "NORMAL" | "RECONCILING";

interface EventItem {
  id: string;
  severity: "Warning" | "Normal" | "Reconciling";
  time: string;
  resource: string;
  icon: "data_object" | "inventory_2" | "database";
  namespace: string;
  message: string;
  status: "Unhealthy" | "Succeeded" | "Reconciling" | "Normal";
}

const INITIAL_EVENTS: EventItem[] = [
  {
    id: "ev1",
    severity: "Warning",
    time: "2m ago",
    resource: "vm-payments-batch",
    icon: "data_object",
    namespace: "default",
    message: "Claim not found in cluster namespace",
    status: "Unhealthy",
  },
  {
    id: "ev2",
    severity: "Warning",
    time: "14m ago",
    resource: "storage-cards",
    icon: "inventory_2",
    namespace: "storage-system",
    message: "Claim not found in cluster namespace",
    status: "Unhealthy",
  },
  {
    id: "ev3",
    severity: "Warning",
    time: "42m ago",
    resource: "pgsql-payments-dev",
    icon: "database",
    namespace: "core-payments",
    message: "Claim not found in cluster namespace",
    status: "Unhealthy",
  },
  {
    id: "ev4",
    severity: "Normal",
    time: "1h ago",
    resource: "pgsql-payments-dev",
    icon: "database",
    namespace: "core-payments",
    message: "Revision 1: SUCCEEDED — Composite resource reconciled",
    status: "Succeeded",
  },
  {
    id: "ev5",
    severity: "Reconciling",
    time: "2h ago",
    resource: "vm-payments-batch",
    icon: "data_object",
    namespace: "default",
    message: "Revision 1: RECONCILING — Applying Terraform / Azure ARM payload",
    status: "Reconciling",
  },
  {
    id: "ev6",
    severity: "Normal",
    time: "3h ago",
    resource: "storage-cards",
    icon: "inventory_2",
    namespace: "storage-system",
    message: "Revision 1: MERGED — Pull request #142 merged by platform-bot",
    status: "Normal",
  },
];

export default function DashboardPage() {
  const [filterQuery, setFilterQuery] = useState("");
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>("ALL");
  const [timeRange, setTimeRange] = useState<TimeRange>("1h");
  const [timeDropdownOpen, setTimeDropdownOpen] = useState(false);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [requestApproved, setRequestApproved] = useState(false);

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
    setTimeout(() => setCopiedId(null), 1800);
  };

  const filteredEvents = useMemo(() => {
    return INITIAL_EVENTS.filter((ev) => {
      if (severityFilter !== "ALL" && ev.severity.toUpperCase() !== severityFilter) {
        return false;
      }
      if (filterQuery.trim()) {
        const q = filterQuery.toLowerCase();
        return (
          ev.resource.toLowerCase().includes(q) ||
          ev.message.toLowerCase().includes(q) ||
          ev.namespace.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [severityFilter, filterQuery]);

  const timeRangeLabels: Record<TimeRange, string> = {
    "15m": "Last 15 minutes",
    "1h": "Last 1 hour",
    "24h": "Last 24 hours",
    "7d": "Last 7 days",
  };

  return (
    <div className="w-full flex flex-col space-y-space-xl pb-12">
      {/* Top Action & Overview Title Bar */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-space-lg animate-fade-slide-up">
        <div className="space-y-space-xs">
          <div className="flex flex-wrap items-center gap-space-sm">
            <h1 className="font-headline-xl text-headline-xl text-on-surface tracking-tight font-semibold">
              Infrastructure Overview
            </h1>
            <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-surface-container-high/70 backdrop-blur-sm text-primary font-label-sm text-label-sm shadow-sm border border-surface-container-highest/60 hover:border-primary/40 transition-colors">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
              </span>
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
        <div className="flex flex-wrap items-center gap-space-sm relative">
          {/* Timeframe Selector Dropdown */}
          <div className="relative">
            <button
              onClick={() => setTimeDropdownOpen((prev) => !prev)}
              className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container transition-all text-on-surface font-label-md text-label-md shadow-sm border border-surface-container-high/60 cursor-pointer active:scale-98"
              type="button"
            >
              <Icon name="schedule" className="h-4 w-4 text-on-surface-variant" />
              <span>{timeRangeLabels[timeRange]}</span>
              <Icon
                name="expand_more"
                className={`h-3.5 w-3.5 text-outline transition-transform duration-200 ${
                  timeDropdownOpen ? "rotate-180" : ""
                }`}
              />
            </button>

            {timeDropdownOpen && (
              <div
                className="absolute right-0 top-full mt-1.5 w-44 rounded-xl bg-surface-container-low border border-surface-container-high shadow-xl py-1 z-30 animate-fade-slide-up"
                onMouseLeave={() => setTimeDropdownOpen(false)}
              >
                {(Object.keys(timeRangeLabels) as TimeRange[]).map((key) => (
                  <button
                    key={key}
                    onClick={() => {
                      setTimeRange(key);
                      setTimeDropdownOpen(false);
                    }}
                    className={`w-full px-3 py-2 text-left text-xs flex items-center justify-between transition-colors ${
                      timeRange === key
                        ? "bg-[#2563eb]/15 text-primary font-medium"
                        : "text-on-surface hover:bg-surface-container"
                    }`}
                  >
                    <span>{timeRangeLabels[key]}</span>
                    {timeRange === key && <Icon name="check" className="h-3.5 w-3.5 text-primary" />}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Refresh Button */}
          <button
            onClick={() => dashboardQuery.refetch()}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container transition-all text-on-surface font-label-md text-label-md shadow-sm border border-surface-container-high/60 cursor-pointer active:scale-98 group"
            type="button"
          >
            <Icon
              name="autorenew"
              className={`h-4 w-4 text-primary group-hover:rotate-180 transition-transform duration-500 ${
                dashboardQuery.isFetching ? "animate-spin" : ""
              }`}
            />
            <span>Refresh (30s)</span>
          </button>

          {/* Request Resource CTA */}
          <Link
            href="/infrastructure"
            className="inline-flex items-center gap-2 px-4 py-1.5 rounded-lg bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white font-label-md text-label-md font-medium shadow-md shadow-blue-500/25 transition-all duration-200 cursor-pointer hover:shadow-lg hover:shadow-blue-500/30 active:scale-98"
          >
            <Icon name="add" className="h-4 w-4 text-white" />
            <span>Request Resource</span>
          </Link>
        </div>
      </div>

      {/* SECTION 1: INFRASTRUCTURE KPIS (6 Cards with Staggered Motion) */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-space-md">
        {/* KPI 1: Total Composed */}
        <div
          className="group p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-primary/50 hover:-translate-y-1 hover:shadow-md transition-all duration-300 animate-fade-slide-up"
          style={{ animationDelay: "50ms" }}
        >
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Total Composed
            </span>
            <div className="w-8 h-8 rounded-lg bg-surface-container-high flex items-center justify-center text-primary group-hover:bg-primary/10 group-hover:scale-110 transition-all duration-200">
              <Icon name="layers" className="h-4 w-4 text-primary" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold tracking-tight">
                {totalComposed}
              </span>
              <span className="font-label-sm text-label-sm px-2 py-0.5 rounded-full bg-surface-container-highest text-primary font-mono border border-primary/20">
                XRD bound
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1.5">Across all namespaces</p>
          </div>
        </div>

        {/* KPI 2: Ready Resources */}
        <div
          className="group p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-emerald-500/50 hover:-translate-y-1 hover:shadow-md transition-all duration-300 animate-fade-slide-up"
          style={{ animationDelay: "100ms" }}
        >
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Ready Resources
            </span>
            <div className="w-8 h-8 rounded-lg bg-surface-container-high flex items-center justify-center text-secondary group-hover:bg-emerald-500/10 group-hover:text-emerald-500 group-hover:scale-110 transition-all duration-200">
              <Icon name="check_circle" className="h-4 w-4" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold tracking-tight">
                {readyCount}
              </span>
              <span className="inline-flex items-center gap-1.5 font-label-sm text-label-sm text-emerald-400 px-2 py-0.5 rounded-full bg-emerald-500/10 border border-emerald-500/20">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                {totalComposed > 0 ? Math.round((readyCount / totalComposed) * 100) : 0}% Ready
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1.5">
              {readyCount} of {totalComposed} reconciled ready
            </p>
          </div>
        </div>

        {/* KPI 3: Not Ready */}
        <div
          className="group p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-error/50 hover:-translate-y-1 hover:shadow-md transition-all duration-300 animate-fade-slide-up"
          style={{ animationDelay: "150ms" }}
        >
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Not Ready
            </span>
            <div className="w-8 h-8 rounded-lg bg-surface-container-high flex items-center justify-center text-error group-hover:bg-error/10 group-hover:scale-110 transition-all duration-200">
              <Icon name="cancel" className="h-4 w-4 text-error" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold tracking-tight">
                {notReadyCount}
              </span>
              <span className="inline-flex items-center gap-1.5 font-label-sm text-label-sm text-error px-2 py-0.5 rounded-full bg-error-container/20 border border-error/20">
                <span className="w-1.5 h-1.5 rounded-full bg-error"></span>
                {notReadyCount} Failed
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1.5">No hard reconciliation errors</p>
          </div>
        </div>

        {/* KPI 4: Unknown Health */}
        <div
          className="group p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-secondary-fixed/50 hover:-translate-y-1 hover:shadow-md transition-all duration-300 animate-fade-slide-up"
          style={{ animationDelay: "200ms" }}
        >
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Unknown Health
            </span>
            <div className="w-8 h-8 rounded-lg bg-surface-container-high flex items-center justify-center text-secondary-fixed group-hover:scale-110 transition-all duration-200">
              <Icon name="help" className="h-4 w-4 text-secondary-fixed" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold tracking-tight">
                {unknownCount}
              </span>
              <span className="inline-flex items-center gap-1 font-label-sm text-label-sm text-secondary px-2 py-0.5 rounded-full bg-surface-container-highest border border-surface-container-high">
                Pending Probes
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1.5 truncate" title="Pending health probe / missing visibility">
              Awaiting probe / XRD read
            </p>
          </div>
        </div>

        {/* KPI 5: Healthy Providers */}
        <div
          className="group p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-primary/50 hover:-translate-y-1 hover:shadow-md transition-all duration-300 animate-fade-slide-up"
          style={{ animationDelay: "250ms" }}
        >
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Healthy Providers
            </span>
            <div className="w-8 h-8 rounded-lg bg-surface-container-high flex items-center justify-center text-primary group-hover:bg-primary/10 group-hover:scale-110 transition-all duration-200">
              <Icon name="hub" className="h-4 w-4 text-primary" />
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold tracking-tight">
                11<span className="text-on-surface-variant text-headline-md font-normal">/13</span>
              </span>
              <span className="font-label-sm text-label-sm text-primary font-mono font-medium">84.6%</span>
            </div>
            <div className="w-full bg-surface-container-highest rounded-full h-1.5 mt-2.5 overflow-hidden">
              <div
                className="bg-gradient-to-r from-blue-500 to-indigo-500 h-full rounded-full transition-all duration-700 ease-out"
                style={{ width: "84.6%" }}
              ></div>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-1.5">2 degraded (Compute, Net)</p>
          </div>
        </div>

        {/* KPI 6: Pending Approvals */}
        <div
          className="group p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-amber-500/50 hover:-translate-y-1 hover:shadow-md transition-all duration-300 animate-fade-slide-up"
          style={{ animationDelay: "300ms" }}
        >
          <div className="flex items-center justify-between">
            <span className="font-label-sm text-label-sm text-on-surface-variant uppercase tracking-wider font-mono">
              Pending Approvals
            </span>
            <div className="relative w-8 h-8 rounded-lg bg-surface-container-high flex items-center justify-center text-secondary-fixed group-hover:scale-110 transition-all duration-200">
              <Icon name="approval" className="h-4 w-4 text-secondary-fixed" />
              <span className="absolute top-0 right-0 w-2.5 h-2.5 rounded-full bg-amber-400 animate-ping"></span>
            </div>
          </div>
          <div className="mt-space-md">
            <div className="flex items-baseline gap-2">
              <span className="font-display-lg text-display-lg text-on-surface font-bold tracking-tight">1</span>
              <span className="font-label-sm text-label-sm text-amber-400 bg-amber-400/10 border border-amber-400/20 px-2 py-0.5 rounded-full">
                Requires review
              </span>
            </div>
            <Link
              className="inline-flex items-center gap-1 font-label-sm text-label-sm text-primary hover:text-blue-400 mt-1.5 font-medium transition-colors group-hover:translate-x-0.5 duration-200"
              href="/infrastructure"
            >
              <span>Review requests</span>
              <Icon name="arrow_forward" className="h-3.5 w-3.5 text-primary" />
            </Link>
          </div>
        </div>
      </div>

      {/* SECTION 2: RESOURCE HEALTH & DISTRIBUTION (3 Cards) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-space-md animate-fade-slide-up" style={{ animationDelay: "200ms" }}>
        {/* Card A: Resource Health Donut */}
        <div className="p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-surface-container-highest transition-all duration-300">
          <div>
            <div className="flex items-center justify-between">
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Resource Health</h2>
              <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-high text-on-surface-variant border border-surface-container-highest/60 font-mono">
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
                  className="text-surface-container-highest/60"
                  cx="50"
                  cy="50"
                  fill="none"
                  r="40"
                  stroke="currentColor"
                  strokeWidth="11"
                />
                <circle
                  className="text-blue-500 animate-donut"
                  cx="50"
                  cy="50"
                  fill="none"
                  r="40"
                  stroke="currentColor"
                  strokeDasharray="251.2"
                  strokeDashoffset="0"
                  strokeLinecap="round"
                  strokeWidth="11"
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                <span className="font-display-lg text-display-lg text-on-surface font-bold leading-none tracking-tight">
                  {totalComposed}
                </span>
                <span className="font-label-sm text-label-sm text-on-surface-variant font-mono mt-1">Total XR</span>
              </div>
            </div>

            {/* Legend */}
            <div className="space-y-2 mt-2">
              <div className="flex items-center justify-between text-body-sm font-body-sm">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-emerald-500"></span>
                  <span className="text-on-surface">Ready</span>
                </div>
                <span className="font-mono text-on-surface-variant">
                  {readyCount} ({totalComposed > 0 ? Math.round((readyCount / totalComposed) * 100) : 0}%)
                </span>
              </div>
              <div className="flex items-center justify-between text-body-sm font-body-sm">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-error"></span>
                  <span className="text-on-surface">Not Ready / Failed</span>
                </div>
                <span className="font-mono text-on-surface-variant">
                  {notReadyCount} ({totalComposed > 0 ? Math.round((notReadyCount / totalComposed) * 100) : 0}%)
                </span>
              </div>
              <div className="flex items-center justify-between text-body-sm font-body-sm">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-blue-500"></span>
                  <span className="text-on-surface">Unknown / Syncing</span>
                </div>
                <span className="font-mono text-primary font-medium">
                  {unknownCount} ({totalComposed > 0 ? Math.round((unknownCount / totalComposed) * 100) : 100}%)
                </span>
              </div>
            </div>
          </div>

          <div className="mt-space-md p-space-sm rounded-lg bg-surface-container font-body-sm text-body-sm text-on-surface-variant flex items-start gap-2 border border-surface-container-high/40">
            <Icon name="info" className="h-4 w-4 text-primary shrink-0 mt-0.5" />
            <span>Health visibility probes awaiting XRD readiness status evaluation.</span>
          </div>
        </div>

        {/* Card B: Cloud Distribution */}
        <div className="p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-surface-container-highest transition-all duration-300">
          <div>
            <div className="flex items-center justify-between">
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Cloud Distribution</h2>
              <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-high text-primary font-mono border border-primary/20">
                1 Target Provider
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Resources managed across registered hyperscalers
            </p>

            <div className="space-y-space-md mt-space-lg">
              {/* Azure Row */}
              <div className="space-y-2">
                <div className="flex items-center justify-between font-body-sm text-body-sm">
                  <div className="flex items-center gap-2">
                    <Icon name="cloud" className="h-4 w-4 text-blue-500" />
                    <span className="font-medium text-on-surface">Microsoft Azure</span>
                  </div>
                  <span className="font-mono text-primary font-medium">3 (100%)</span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden">
                  <div
                    className="bg-gradient-to-r from-blue-500 to-indigo-600 h-full rounded-full transition-all duration-700 ease-out"
                    style={{ width: "100%" }}
                  ></div>
                </div>
                <span className="font-label-sm text-label-sm text-outline font-mono">sub: prod-hybrid-mesh-01</span>
              </div>

              {/* AWS Row */}
              <div className="space-y-2 opacity-60">
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
              <div className="space-y-2 opacity-60">
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

          <div className="mt-space-md p-space-sm rounded-lg bg-surface-container font-body-sm text-body-sm text-on-surface-variant flex items-center gap-2 border border-surface-container-high/40">
            <Icon name="check" className="h-4 w-4 text-emerald-500" />
            <span>Current active footprint concentrated in Azure East US 2.</span>
          </div>
        </div>

        {/* Card C: Environment Distribution */}
        <div className="p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between hover:border-surface-container-highest transition-all duration-300">
          <div>
            <div className="flex items-center justify-between">
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Environment Distribution</h2>
              <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-high text-primary font-mono border border-primary/20">
                1 active
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Logical partition topology for current deployment
            </p>

            <div className="space-y-space-md mt-space-lg">
              {/* Dev */}
              <div className="p-space-md rounded-xl bg-surface-container space-y-2 border border-surface-container-high/50 hover:border-primary/40 transition-colors">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2.5 h-2.5 rounded-sm bg-blue-500"></span>
                    <span className="font-label-md text-label-md font-semibold text-on-surface">Development (dev)</span>
                  </div>
                  <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-[#2563eb] text-white font-mono shadow-sm">
                    {totalComposed} resources
                  </span>
                </div>
                <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden">
                  <div
                    className="bg-gradient-to-r from-blue-500 to-indigo-500 h-full rounded-full transition-all duration-700 ease-out"
                    style={{ width: "100%" }}
                  ></div>
                </div>
                <p className="font-label-sm text-label-sm text-on-surface-variant">
                  Primary active staging lane for squad platform tests
                </p>
              </div>

              {/* Staging */}
              <div className="p-space-md rounded-xl bg-surface-container-low opacity-60 space-y-1.5 border border-surface-container-high/20">
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
              <div className="p-space-md rounded-xl bg-surface-container-low opacity-60 space-y-1.5 border border-surface-container-high/20">
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
            <span className="font-mono text-primary font-medium">Role: platform-admin</span>
          </div>
        </div>
      </div>

      {/* SECTION 3: CROSSPLANE PLATFORM HEALTH */}
      <div
        className="p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 space-y-space-md animate-fade-slide-up"
        style={{ animationDelay: "300ms" }}
      >
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-space-sm">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                <Icon name="memory" className="h-4 w-4 text-primary" />
              </div>
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">
                Crossplane Control Plane
              </h2>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Control plane reconciliation loop status and core package controllers
            </p>
          </div>
          <Link
            className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:text-blue-400 font-medium transition-colors"
            href="/platform"
          >
            <span>View platform details</span>
            <Icon name="arrow_forward" className="h-4 w-4" />
          </Link>
        </div>

        {/* Metric Pills Row */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-space-sm">
          <div className="p-space-sm rounded-lg bg-surface-container flex items-center justify-between border border-surface-container-high/40 hover:border-primary/40 transition-colors">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-secondary-fixed"></span>
              <span className="font-body-sm text-body-sm text-on-surface">Providers</span>
            </div>
            <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-high text-secondary-fixed font-mono border border-surface-container-highest">
              11 / 13 healthy
            </span>
          </div>
          <div className="p-space-sm rounded-lg bg-surface-container flex items-center justify-between border border-surface-container-high/40 hover:border-primary/40 transition-colors">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-outline"></span>
              <span className="font-body-sm text-body-sm text-on-surface">Functions</span>
            </div>
            <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-high text-outline font-mono border border-surface-container-highest">
              0 installed
            </span>
          </div>
          <div className="p-space-sm rounded-lg bg-surface-container flex items-center justify-between border border-surface-container-high/40 hover:border-primary/40 transition-colors">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-secondary-fixed"></span>
              <span className="font-body-sm text-body-sm text-on-surface">Compositions</span>
            </div>
            <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-high text-on-surface-variant font-mono border border-surface-container-highest">
              Unavailable (RBAC)
            </span>
          </div>
          <div className="p-space-sm rounded-lg bg-surface-container flex items-center justify-between border border-surface-container-high/40 hover:border-primary/40 transition-colors">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-outline"></span>
              <span className="font-body-sm text-body-sm text-on-surface">XRDs</span>
            </div>
            <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-high text-on-surface-variant font-mono border border-surface-container-highest">
              Awaiting sync
            </span>
          </div>
        </div>

        {/* Health Progress Bar */}
        <div className="space-y-1.5 pt-1">
          <div className="flex items-center justify-between font-label-sm text-label-sm">
            <span className="text-on-surface-variant">Overall Provider Reconciliation Health</span>
            <span className="font-mono text-on-surface font-semibold">84.6% Operational</span>
          </div>
          <div className="w-full bg-surface-container-highest rounded-full h-2 overflow-hidden flex">
            <div className="bg-gradient-to-r from-blue-500 to-indigo-600 h-full" style={{ width: "84.6%" }}></div>
            <div className="bg-error h-full" style={{ width: "15.4%" }}></div>
          </div>
        </div>

        {/* Operational Issues List */}
        <div className="overflow-x-auto rounded-lg border border-surface-container-high/40">
          <table className="w-full text-left font-body-sm text-body-sm">
            <thead>
              <tr className="text-outline uppercase font-mono text-[11px] tracking-wider bg-surface-container border-b border-surface-container-high/60">
                <th className="py-2.5 px-space-md">Provider Package</th>
                <th className="py-2.5 px-space-md">Version</th>
                <th className="py-2.5 px-space-md">Reconciliation Condition</th>
                <th className="py-2.5 px-space-md">Error Diagnostic</th>
                <th className="py-2.5 px-space-md text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-container/40">
              <tr className="hover:bg-surface-container/50 transition-colors">
                <td className="py-space-sm px-space-md">
                  <div className="flex items-center gap-2 font-mono font-medium text-on-surface">
                    <Icon name="deployed_code" className="h-4 w-4 text-error shrink-0" />
                    <span>provider-azure-compute</span>
                  </div>
                </td>
                <td className="py-space-sm px-space-md font-mono text-on-surface-variant">v0.42.1</td>
                <td className="py-space-sm px-space-md">
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-error-container/30 text-error font-label-sm text-label-sm font-semibold border border-error/20">
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
                    className="px-3 py-1 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-sm text-label-sm transition-all duration-200 inline-block border border-surface-container-highest/60 hover:border-primary/40 active:scale-95"
                  >
                    Inspect Details
                  </Link>
                </td>
              </tr>
              <tr className="hover:bg-surface-container/50 transition-colors">
                <td className="py-space-sm px-space-md">
                  <div className="flex items-center gap-2 font-mono font-medium text-on-surface">
                    <Icon name="deployed_code" className="h-4 w-4 text-error shrink-0" />
                    <span>provider-azure-network</span>
                  </div>
                </td>
                <td className="py-space-sm px-space-md font-mono text-on-surface-variant">v0.41.0</td>
                <td className="py-space-sm px-space-md">
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-error-container/30 text-error font-label-sm text-label-sm font-semibold border border-error/20">
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
                    className="px-3 py-1 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-sm text-label-sm transition-all duration-200 inline-block border border-surface-container-highest/60 hover:border-primary/40 active:scale-95"
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
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-space-md animate-fade-slide-up" style={{ animationDelay: "400ms" }}>
        {/* Left Column: Needs Attention (7 cols) */}
        <div className="lg:col-span-7 p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between space-y-space-md hover:border-surface-container-highest transition-all duration-300">
          <div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="w-7 h-7 rounded-lg bg-error/10 flex items-center justify-center text-error">
                  <Icon name="notification_important" className="h-4 w-4 text-error" />
                </div>
                <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Needs Attention</h2>
              </div>
              <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-error-container/30 text-error font-mono font-semibold border border-error/20">
                4 items
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Priority alerts and configuration anomalies impacting reconciliation
            </p>

            <div className="space-y-space-sm mt-space-md">
              {/* Item 1: Critical */}
              <div className="p-space-sm rounded-xl bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 hover:bg-surface-container-high/70 transition-all duration-200 border border-surface-container-high/40">
                <div className="flex items-start gap-3 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-2 py-0.5 rounded-md bg-error-container text-white uppercase font-mono font-bold shadow-sm">
                    Critical
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-semibold text-on-surface truncate">
                      provider-azure-compute
                    </p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      Provider pod reconciliation failing in backoff loop
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                  <Link
                    href="/platform?tab=Providers"
                    className="font-label-sm text-label-sm text-primary hover:text-blue-400 font-medium hover:underline"
                  >
                    Resolve
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link
                    href="/platform?tab=Providers"
                    className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface"
                  >
                    View
                  </Link>
                </div>
              </div>

              {/* Item 2: Warning */}
              <div className="p-space-sm rounded-xl bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 hover:bg-surface-container-high/70 transition-all duration-200 border border-surface-container-high/40">
                <div className="flex items-start gap-3 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-2 py-0.5 rounded-md bg-amber-500/20 text-amber-400 uppercase font-mono font-bold border border-amber-500/30">
                    Warning
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-semibold text-on-surface truncate">vm-payments-batch</p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      Missing resource health probe configuration from XRD definition
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                  <Link
                    href="/infrastructure"
                    className="font-label-sm text-label-sm text-primary hover:text-blue-400 font-medium hover:underline"
                  >
                    Resolve
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link
                    href="/infrastructure"
                    className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface"
                  >
                    View
                  </Link>
                </div>
              </div>

              {/* Item 3: Warning */}
              <div className="p-space-sm rounded-xl bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 hover:bg-surface-container-high/70 transition-all duration-200 border border-surface-container-high/40">
                <div className="flex items-start gap-3 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-2 py-0.5 rounded-md bg-amber-500/20 text-amber-400 uppercase font-mono font-bold border border-amber-500/30">
                    Warning
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-semibold text-on-surface truncate">storage-cards</p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      Claim not found in target cluster namespace: storage-system
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                  <Link
                    href="/infrastructure"
                    className="font-label-sm text-label-sm text-primary hover:text-blue-400 font-medium hover:underline"
                  >
                    Resolve
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link
                    href="/infrastructure"
                    className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface"
                  >
                    View
                  </Link>
                </div>
              </div>

              {/* Item 4: Info */}
              <div className="p-space-sm rounded-xl bg-surface-container flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 hover:bg-surface-container-high/70 transition-all duration-200 border border-surface-container-high/40">
                <div className="flex items-start gap-3 min-w-0">
                  <span className="mt-0.5 font-label-sm text-label-sm px-2 py-0.5 rounded-md bg-blue-500/20 text-primary uppercase font-mono font-bold border border-blue-500/30">
                    Info
                  </span>
                  <div className="min-w-0">
                    <p className="font-mono text-body-sm font-semibold text-on-surface truncate">pgsql-payments-dev</p>
                    <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                      PostgreSQL Server binding established, awaiting consumer secret export
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                  <Link
                    href="/infrastructure"
                    className="font-label-sm text-label-sm text-primary hover:text-blue-400 font-medium hover:underline"
                  >
                    Export
                  </Link>
                  <span className="text-outline text-xs">•</span>
                  <Link
                    href="/infrastructure"
                    className="font-label-sm text-label-sm text-on-surface-variant hover:text-on-surface"
                  >
                    View
                  </Link>
                </div>
              </div>
            </div>
          </div>

          <div className="pt-2 flex items-center justify-between font-label-sm text-label-sm text-on-surface-variant">
            <span>Active filter: Severity ≥ Warning</span>
            <Link className="text-primary hover:underline hover:text-blue-400 transition-colors" href="/infrastructure">
              Configure alert routing →
            </Link>
          </div>
        </div>

        {/* Right Column: Active Provisioning Requests (5 cols) */}
        <div className="lg:col-span-5 p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 flex flex-col justify-between space-y-space-md hover:border-surface-container-highest transition-all duration-300">
          <div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                  <Icon name="published_with_changes" className="h-4 w-4 text-primary" />
                </div>
                <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">
                  Active Provisioning Requests
                </h2>
              </div>
              <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-secondary-container text-secondary-fixed font-mono font-medium border border-surface-container-highest">
                1 Active
              </span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Pending approval gates and progressive rollout stages
            </p>

            {/* Request Card */}
            <div className="mt-space-md p-space-md rounded-xl bg-surface-container space-y-space-md border border-surface-container-high/50 hover:border-primary/40 transition-colors shadow-sm">
              <div className="flex items-start justify-between">
                <div>
                  <span className="font-headline-md text-headline-md text-on-surface font-semibold font-mono">
                    vm-payments-batch
                  </span>
                  <div className="flex items-center gap-2 mt-1 font-label-sm text-label-sm text-on-surface-variant">
                    <span>
                      Type: <strong className="text-on-surface font-mono">VirtualMachine</strong>
                    </span>
                    <span>•</span>
                    <span>
                      Env: <strong className="text-on-surface">Dev</strong>
                    </span>
                  </div>
                </div>
                <span
                  className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full font-label-sm text-label-sm font-semibold border ${
                    requestApproved
                      ? "bg-emerald-500/20 text-emerald-400 border-emerald-500/30"
                      : "bg-amber-500/20 text-amber-400 border-amber-500/30"
                  }`}
                >
                  <span
                    className={`w-1.5 h-1.5 rounded-full ${
                      requestApproved ? "bg-emerald-400" : "bg-amber-400 animate-ping"
                    }`}
                  ></span>
                  {requestApproved ? "APPROVED" : "WAITING_APPROVAL"}
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
              <div className="space-y-2 pt-2">
                <span className="font-label-sm text-label-sm text-outline uppercase font-mono tracking-wider">
                  Lifecycle Pipeline
                </span>
                <div className="grid grid-cols-5 gap-1.5 text-center font-label-sm text-label-sm">
                  {/* Step 1: Requested */}
                  <div className="space-y-1">
                    <div className="h-1.5 rounded-full bg-blue-500"></div>
                    <span className="text-on-surface font-medium truncate block text-[10px]">Req</span>
                  </div>
                  {/* Step 2: Validating */}
                  <div className="space-y-1">
                    <div className="h-1.5 rounded-full bg-blue-500"></div>
                    <span className="text-on-surface font-medium truncate block text-[10px]">Valid</span>
                  </div>
                  {/* Step 3: Approval */}
                  <div className="space-y-1">
                    <div
                      className={`h-1.5 rounded-full ${
                        requestApproved ? "bg-emerald-400" : "bg-amber-400 animate-pulse"
                      }`}
                    ></div>
                    <span
                      className={`font-bold truncate block text-[10px] ${
                        requestApproved ? "text-emerald-400" : "text-amber-400"
                      }`}
                    >
                      Approve
                    </span>
                  </div>
                  {/* Step 4: Provisioning */}
                  <div className="space-y-1">
                    <div
                      className={`h-1.5 rounded-full ${
                        requestApproved ? "bg-blue-400 animate-pulse" : "bg-surface-container-highest"
                      }`}
                    ></div>
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
                  onClick={() => setRequestApproved(true)}
                  disabled={requestApproved}
                  className={`flex-1 py-1.5 px-3 rounded-lg font-label-md text-label-md font-medium transition-all text-center shadow-sm cursor-pointer active:scale-98 ${
                    requestApproved
                      ? "bg-emerald-600 text-white cursor-default"
                      : "bg-[#2563eb] text-white hover:bg-blue-600 hover:shadow-blue-500/25"
                  }`}
                  type="button"
                >
                  {requestApproved ? "Approved ✓" : "Approve Request"}
                </button>
                <button
                  onClick={() => setRequestApproved(false)}
                  className="py-1.5 px-3 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-error font-label-md text-label-md transition-colors cursor-pointer active:scale-98"
                  type="button"
                >
                  Reject
                </button>
                <Link
                  href="/infrastructure"
                  className="py-1.5 px-2.5 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-md text-label-md transition-colors inline-flex items-center justify-center active:scale-98"
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

      {/* SECTION 5: RECENT INFRASTRUCTURE ACTIVITY (Full Interactive Table) */}
      <div
        className="p-space-lg rounded-xl bg-surface-container-low/90 backdrop-blur-sm shadow-sm border border-surface-container-high/60 space-y-space-md animate-fade-slide-up"
        style={{ animationDelay: "500ms" }}
      >
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-space-md">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                <Icon name="history" className="h-4 w-4 text-primary" />
              </div>
              <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">
                Recent Events &amp; Activity
              </h2>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Real-time audit log of Crossplane composition reconciliation iterations
            </p>
          </div>
          <Link
            className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:text-blue-400 font-medium transition-colors"
            href="/infrastructure"
          >
            <span>View all 128 events</span>
            <Icon name="arrow_forward" className="h-4 w-4" />
          </Link>
        </div>

        {/* Controls & Filter Toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-space-sm pt-1">
          <div className="flex flex-wrap items-center gap-2 flex-1 max-w-xl">
            <div className="relative flex-1 min-w-[220px]">
              <div className="absolute left-3 top-1/2 -translate-y-1/2 text-outline pointer-events-none">
                <Icon name="search" className="h-4 w-4 text-outline" />
              </div>
              <input
                value={filterQuery}
                onChange={(e) => setFilterQuery(e.target.value)}
                className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-surface-container text-on-surface placeholder:text-outline font-body-sm text-body-sm focus:outline-none focus:ring-1 focus:ring-primary shadow-inner border border-surface-container-high/60 transition-colors"
                placeholder="Filter events by name, message..."
                type="text"
              />
            </div>

            {/* Severity Filter Tabs */}
            <div className="flex items-center gap-1 bg-surface-container p-1 rounded-lg border border-surface-container-high/60">
              {(["ALL", "WARNING", "NORMAL", "RECONCILING"] as SeverityFilter[]).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setSeverityFilter(tab)}
                  className={`px-2.5 py-0.5 rounded-md text-xs font-medium transition-all ${
                    severityFilter === tab
                      ? "bg-[#2563eb] text-white shadow-sm"
                      : "text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high"
                  }`}
                  type="button"
                >
                  {tab === "ALL" ? "All" : tab.charAt(0) + tab.slice(1).toLowerCase()}
                </button>
              ))}
            </div>
          </div>

          <div className="flex items-center gap-1 text-outline font-label-sm text-label-sm">
            <span>Showing {filteredEvents.length} items</span>
          </div>
        </div>

        {/* Dense Table */}
        <div className="overflow-x-auto rounded-lg border border-surface-container-high/50">
          <table className="w-full text-left font-body-sm text-body-sm">
            <thead>
              <tr className="text-outline uppercase font-mono text-[11px] tracking-wider bg-surface-container border-b border-surface-container-high/60">
                <th className="py-2.5 px-space-md">Severity</th>
                <th className="py-2.5 px-space-md">Time</th>
                <th className="py-2.5 px-space-md">Resource</th>
                <th className="py-2.5 px-space-md">Namespace</th>
                <th className="py-2.5 px-space-md">Event Message</th>
                <th className="py-2.5 px-space-md">Status</th>
                <th className="py-2.5 px-space-md text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-container/30 font-code-sm">
              {filteredEvents.map((ev) => (
                <tr key={ev.id} className="hover:bg-surface-container/60 transition-colors">
                  <td className="py-2.5 px-space-md whitespace-nowrap">
                    <span
                      className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full font-label-sm text-label-sm font-medium border ${
                        ev.severity === "Warning"
                          ? "bg-amber-500/15 text-amber-400 border-amber-500/20"
                          : ev.severity === "Reconciling"
                          ? "bg-blue-500/15 text-primary border-blue-500/20"
                          : "bg-surface-container-highest text-secondary border-surface-container-high"
                      }`}
                    >
                      <span
                        className={`w-1.5 h-1.5 rounded-full ${
                          ev.severity === "Warning"
                            ? "bg-amber-400"
                            : ev.severity === "Reconciling"
                            ? "bg-primary animate-pulse"
                            : "bg-secondary"
                        }`}
                      ></span>
                      {ev.severity}
                    </span>
                  </td>
                  <td className="py-2.5 px-space-md text-on-surface-variant whitespace-nowrap font-mono text-[11px]">
                    {ev.time}
                  </td>
                  <td className="py-2.5 px-space-md font-mono text-on-surface font-medium whitespace-nowrap">
                    <div className="flex items-center gap-1.5">
                      <Icon name={ev.icon} className="h-4 w-4 text-primary shrink-0" />
                      <span>{ev.resource}</span>
                    </div>
                  </td>
                  <td className="py-2.5 px-space-md font-mono text-on-surface-variant">{ev.namespace}</td>
                  <td className="py-2.5 px-space-md text-on-surface truncate max-w-md">{ev.message}</td>
                  <td className="py-2.5 px-space-md whitespace-nowrap">
                    <span className="font-label-sm text-label-sm px-2.5 py-0.5 rounded-full bg-surface-container-highest text-secondary-fixed border border-surface-container-high">
                      {ev.status}
                    </span>
                  </td>
                  <td className="py-2.5 px-space-md text-right whitespace-nowrap">
                    <Link
                      href="/infrastructure"
                      className="p-1 rounded-md hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors inline-block"
                      title="Inspect"
                    >
                      <Icon name="open_in_new" className="h-4 w-4" />
                    </Link>
                    <button
                      onClick={() => handleCopy(ev.resource, ev.id)}
                      className="p-1 rounded-md hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface transition-colors cursor-pointer ml-1"
                      title={copiedId === ev.id ? "Copied!" : "Copy Manifest"}
                      type="button"
                    >
                      <Icon
                        name={copiedId === ev.id ? "check" : "content_copy"}
                        className={`h-4 w-4 transition-transform duration-200 ${
                          copiedId === ev.id ? "text-emerald-400 scale-110" : ""
                        }`}
                      />
                    </button>
                  </td>
                </tr>
              ))}

              {filteredEvents.length === 0 && (
                <tr>
                  <td colSpan={7} className="py-8 text-center text-on-surface-variant font-body-sm">
                    No activity found matching the current filters.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Footer Pagination */}
        <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-space-sm font-label-sm text-label-sm text-on-surface-variant">
          <span>
            Showing {filteredEvents.length} of 128 events • Page 1 of 22
          </span>
          <div className="flex items-center gap-1.5">
            <button
              className="px-3 py-1 rounded-lg bg-surface-container text-outline opacity-50 cursor-not-allowed border border-surface-container-high/40"
              disabled
              type="button"
            >
              Previous
            </button>
            <button
              className="px-3 py-1 rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface transition-all duration-150 border border-surface-container-high/60 cursor-pointer active:scale-95"
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
