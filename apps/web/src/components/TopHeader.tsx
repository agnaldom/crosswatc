"use client";

import { useQuery } from "@tanstack/react-query";
import { usePathname, useSearchParams } from "next/navigation";
import { GlobalSearch } from "@/components/GlobalSearch";
import { getContexts } from "@/lib/api";

export function TopHeader() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const kind = searchParams.get("kind");

  const contextsQuery = useQuery({
    queryKey: ["contexts"],
    queryFn: getContexts,
    staleTime: 30_000,
  });

  const currentContext = contextsQuery.data?.currentContext || "crossplane-prod-control-plane";

  let sectionName = "Overview";
  let pageTitle = "Dashboard";

  if (pathname === "/dashboard" || pathname === "/") {
    sectionName = "Overview";
    pageTitle = "Dashboard";
  } else if (pathname.startsWith("/platform")) {
    sectionName = "Infrastructure Catalog";
    pageTitle = "Providers";
  } else if (pathname === "/settings/users") {
    sectionName = "Administration";
    pageTitle = "User Management";
  } else if (pathname === "/settings/contexts") {
    sectionName = "Administration";
    pageTitle = "Context Settings";
  } else if (pathname.startsWith("/infrastructure")) {
    sectionName = "Resources";
    pageTitle = kind || "Crossplane Dashboard";
  }

  return (
    <header className="cw-top-header fixed top-0 left-60 right-0 h-14 bg-surface/80 backdrop-blur-xl z-40 flex items-center justify-between px-space-lg shadow-[0_1px_8px_rgba(0,0,0,0.04)] border-b border-surface-container transition-all duration-200">
      {/* Left: Breadcrumbs & Cluster Context Pill */}
      <div className="flex items-center gap-space-md min-w-0">
        <div className="flex items-center gap-space-xs font-body-md text-body-md truncate">
          <span className="text-on-surface-variant font-medium">{sectionName}</span>
          <span className="text-outline text-xs">/</span>
          <h1 className="text-on-surface font-semibold truncate text-body-md">
            {pageTitle}
          </h1>
        </div>

        <div className="hidden sm:flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-surface-container font-mono text-[11px] text-on-surface-variant shrink-0 border border-surface-container-high/40">
          <span className="material-symbols-outlined text-[13px] text-primary">dns</span>
          <span>ctx: {currentContext}</span>
        </div>
      </div>

      {/* Right: Search, Live Sync, Notifications, Help & Avatar */}
      <div className="flex items-center gap-space-md shrink-0">
        <GlobalSearch />

        <div className="hidden md:flex items-center gap-1.5 text-on-surface-variant font-label-sm text-label-sm">
          <span className="material-symbols-outlined text-[16px] text-primary animate-spin">sync</span>
          <span className="text-outline">Updated 12s ago</span>
        </div>

        <button
          className="relative p-1.5 rounded hover:bg-surface-container hover:text-on-surface text-on-surface-variant transition-colors cursor-pointer"
          type="button"
          aria-label="Notifications"
          title="Notifications"
        >
          <span className="material-symbols-outlined text-[20px]">notifications</span>
          <span className="absolute top-1 right-1 h-3.5 min-w-[14px] px-1 flex items-center justify-center rounded-full bg-error text-on-error font-label-sm text-[9px] leading-none font-bold">
            3
          </span>
        </button>

        <a
          className="p-1.5 rounded hover:bg-surface-container hover:text-on-surface text-on-surface-variant transition-colors cursor-pointer"
          data-path="docs"
          href="https://docs.crossplane.io"
          target="_blank"
          rel="noreferrer"
          title="Crossplane Documentation"
        >
          <span className="material-symbols-outlined text-[20px]">help</span>
        </a>

        <div
          className="w-8 h-8 rounded-full bg-primary flex items-center justify-center cursor-pointer text-on-primary ring-2 ring-surface shadow-sm"
          title="User Profile"
        >
          <span className="material-symbols-outlined text-[18px]">person</span>
        </div>
      </div>
    </header>
  );
}
