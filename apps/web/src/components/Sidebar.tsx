"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { Icon } from "@/components/Icon";
import { api, getContexts } from "@/lib/api";
import { SIDEBAR_KEY } from "@/lib/theme";
import { useTheme } from "@/lib/use-theme";

export function Sidebar() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const currentTab = searchParams.get("tab");
  const currentView = searchParams.get("view");

  const meQuery = useQuery({
    queryKey: ["me"],
    queryFn: api.me,
    staleTime: 60_000,
  });

  const contextsQuery = useQuery({
    queryKey: ["contexts"],
    queryFn: getContexts,
    staleTime: 30_000,
  });

  const dashboardQuery = useQuery({
    queryKey: ["resources", "dashboard"],
    queryFn: api.dashboard,
    staleTime: 30_000,
  });

  const { theme, toggle: toggleTheme } = useTheme();

  const toggleSidebar = () => {
    const d = document.documentElement;
    const collapsed = d.dataset.sidebar !== "collapsed";
    if (collapsed) d.dataset.sidebar = "collapsed";
    else delete d.dataset.sidebar;
    localStorage.setItem(SIDEBAR_KEY, collapsed ? "collapsed" : "expanded");
  };

  const userName = meQuery.data?.user && meQuery.data.user !== "admin" ? meQuery.data.user : "Alex Rivera";
  const userRole = meQuery.data?.sre ? "Principal Platform Eng" : "Principal Platform Eng";
  const currentContext = contextsQuery.data?.currentContext || "k8s-control-plane-us-east";
  const totalResources = dashboardQuery.data?.total ?? 3;

  const isDashboard = pathname === "/dashboard" || pathname === "/";
  const isProviders = pathname === "/platform" && currentTab === "Providers";
  const isFunctions = pathname === "/platform" && currentTab === "Functions";
  const isConfigurations = pathname === "/platform" && currentTab === "Configurations";
  const isCompositions = pathname === "/platform" && currentTab === "Compositions";
  const isXRDs = pathname === "/platform" && currentTab === "XRDs";
  const isComposites = pathname === "/infrastructure" && currentView === "composites";
  const isClaims = pathname === "/infrastructure" && currentView === "claims";
  const isManaged = pathname === "/infrastructure" && currentView === "managed";
  const isUsers = pathname === "/settings/users";
  const isContexts = pathname === "/settings/contexts";

  const getLinkClasses = (isActive: boolean) =>
    isActive
      ? "flex items-center justify-between px-space-sm py-1.5 rounded transition-colors bg-[#2563eb] text-white font-medium text-body-md shadow-sm"
      : "flex items-center justify-between px-space-sm py-1.5 rounded text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors font-body-md text-body-md";

  return (
    <aside className="cw-sidebar fixed left-0 top-0 h-screen w-60 bg-surface-container-low flex flex-col z-50 select-none shadow-[0_1px_8px_rgba(0,0,0,0.04)] border-r border-surface-container transition-all duration-200">
      {/* Top Header */}
      <div className="h-14 px-space-lg flex items-center justify-between border-b border-surface-container/50">
        <Link href="/dashboard" className="flex items-center gap-space-sm min-w-0">
          <div className="w-7 h-7 rounded bg-[#2563eb] flex items-center justify-center text-white shrink-0 shadow-sm">
            <Icon name="deployed_code" className="h-4 w-4 text-white" />
          </div>
          <span className="font-headline-md text-headline-md tracking-tight text-on-surface font-semibold cw-sidebar-hide">
            Crosswatc
          </span>
        </Link>
        <span className="font-label-sm text-label-sm px-space-xs py-0.5 rounded bg-surface-container-highest text-on-surface-variant font-mono cw-sidebar-hide">
          v1.4.2
        </span>
      </div>

      {/* Cluster Context Switcher Pill */}
      <div className="px-space-md py-space-xs cw-sidebar-hide">
        <Link
          href="/settings/contexts"
          className="w-full px-space-sm py-space-xs rounded bg-surface-container flex items-center justify-between text-left hover:bg-surface-container-high transition-colors cursor-pointer border border-surface-container-high/40"
          title="Switch Kubernetes Context"
        >
          <div className="flex items-center gap-space-sm min-w-0">
            <span className="relative flex h-2 w-2 flex-shrink-0">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
            </span>
            <div className="min-w-0">
              <p className="font-label-sm text-label-sm text-on-surface truncate font-mono leading-none">
                {currentContext}
              </p>
              <p className="font-label-sm text-label-sm text-on-surface-variant text-[10px] leading-tight mt-0.5">
                Live Connected
              </p>
            </div>
          </div>
          <Icon name="unfold_more" className="h-3.5 w-3.5 text-on-surface-variant shrink-0" />
        </Link>
      </div>

      {/* Nav Groups Container */}
      <div className="flex-1 overflow-y-auto px-space-sm py-space-xs space-y-space-md">
        {/* Overview */}
        <nav className="space-y-0.5">
          <div className="px-space-sm py-1 font-label-sm text-label-sm text-outline uppercase tracking-wider cw-sidebar-hide">
            Overview
          </div>
          <Link href="/dashboard" className={getLinkClasses(isDashboard)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="grid_view" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Dashboard</span>
            </div>
            {isDashboard && <span className="h-1.5 w-1.5 rounded-full bg-white cw-sidebar-hide"></span>}
          </Link>
        </nav>

        {/* Infrastructure Catalog */}
        <nav className="space-y-0.5">
          <div className="px-space-sm py-1 font-label-sm text-label-sm text-outline uppercase tracking-wider cw-sidebar-hide">
            Infrastructure Catalog
          </div>
          <Link href="/platform?tab=Providers" className={getLinkClasses(isProviders)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="hub" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Providers</span>
            </div>
            <span className="font-label-sm text-label-sm px-1.5 py-0.2 rounded bg-surface-container-highest text-on-surface-variant cw-sidebar-hide">
              13
            </span>
          </Link>
          <Link href="/platform?tab=Functions" className={getLinkClasses(isFunctions)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="code_blocks" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Functions</span>
            </div>
            <span className="font-label-sm text-label-sm px-1.5 py-0.2 rounded bg-surface-container-highest text-on-surface-variant cw-sidebar-hide">
              0
            </span>
          </Link>
          <Link href="/platform?tab=Configurations" className={getLinkClasses(isConfigurations)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="tune" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Configurations</span>
            </div>
          </Link>
          <Link href="/platform?tab=Compositions" className={getLinkClasses(isCompositions)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="schema" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Compositions</span>
            </div>
          </Link>
          <Link href="/platform?tab=XRDs" className={getLinkClasses(isXRDs)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="account_tree" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">XRDs</span>
            </div>
          </Link>
        </nav>

        {/* Resources */}
        <nav className="space-y-0.5">
          <div className="px-space-sm py-1 font-label-sm text-label-sm text-outline uppercase tracking-wider cw-sidebar-hide">
            Resources
          </div>
          <Link href="/infrastructure?view=composites" className={getLinkClasses(isComposites)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="layers" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Composite Resources</span>
            </div>
            <span className="font-label-sm text-label-sm px-1.5 py-0.2 rounded bg-surface-container-highest text-on-surface-variant cw-sidebar-hide">
              {totalResources}
            </span>
          </Link>
          <Link href="/infrastructure?view=claims" className={getLinkClasses(isClaims)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="assignment" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Claims</span>
            </div>
            <span className="font-label-sm text-label-sm px-1.5 py-0.2 rounded bg-surface-container-highest text-on-surface-variant cw-sidebar-hide">
              {totalResources}
            </span>
          </Link>
          <Link href="/infrastructure?view=managed" className={getLinkClasses(isManaged)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="dns" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Managed Resources</span>
            </div>
          </Link>
        </nav>

        {/* Operations */}
        <nav className="space-y-0.5">
          <div className="px-space-sm py-1 font-label-sm text-label-sm text-outline uppercase tracking-wider cw-sidebar-hide">
            Operations
          </div>
          <Link
            href="/infrastructure"
            className="flex items-center justify-between px-space-sm py-1.5 rounded text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors font-body-md text-body-md"
          >
            <div className="flex items-center gap-space-sm">
              <Icon name="approval" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Requests &amp; Approvals</span>
            </div>
            <span className="font-label-sm text-label-sm px-1.5 py-0.2 rounded bg-secondary-container text-secondary-fixed cw-sidebar-hide font-mono">
              1
            </span>
          </Link>
          <Link
            href="/infrastructure"
            className="flex items-center gap-space-sm px-space-sm py-1.5 rounded text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors font-body-md text-body-md"
          >
            <Icon name="history" className="h-4 w-4 shrink-0" />
            <span className="cw-sidebar-hide">Revisions</span>
          </Link>
          <Link
            href="/infrastructure"
            className="flex items-center gap-space-sm px-space-sm py-1.5 rounded text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors font-body-md text-body-md"
          >
            <Icon name="stream" className="h-4 w-4 shrink-0" />
            <span className="cw-sidebar-hide">Events &amp; Activity</span>
          </Link>
        </nav>

        {/* Administration */}
        <nav className="space-y-0.5">
          <div className="px-space-sm py-1 font-label-sm text-label-sm text-outline uppercase tracking-wider cw-sidebar-hide">
            Administration
          </div>
          <Link href="/settings/users" className={getLinkClasses(isUsers)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="group" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">User Management</span>
            </div>
          </Link>
          <Link
            href="/settings/users"
            className="flex items-center gap-space-sm px-space-sm py-1.5 rounded text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors font-body-md text-body-md"
          >
            <Icon name="security" className="h-4 w-4 shrink-0" />
            <span className="cw-sidebar-hide">RBAC &amp; Permissions</span>
          </Link>
          <Link href="/settings/contexts" className={getLinkClasses(isContexts)}>
            <div className="flex items-center gap-space-sm">
              <Icon name="settings_input_component" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Context Settings</span>
            </div>
          </Link>
          <button
            type="button"
            onClick={toggleTheme}
            className="w-full flex items-center justify-between px-space-sm py-1.5 rounded text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors font-body-md text-body-md text-left cursor-pointer"
            title={`Toggle Theme (current: ${theme})`}
          >
            <div className="flex items-center gap-space-sm">
              <Icon name="palette" className="h-4 w-4 shrink-0" />
              <span className="cw-sidebar-hide">Appearance</span>
            </div>
            <span className="font-label-sm text-label-sm px-1.5 py-0.5 rounded bg-surface-container-highest text-on-surface-variant cw-sidebar-hide capitalize">
              {theme}
            </span>
          </button>
        </nav>
      </div>

      {/* Bottom Profile and Collapse Rail */}
      <div className="p-space-xs space-y-space-xs border-t border-surface-container/60">
        <button
          onClick={toggleSidebar}
          className="w-full flex items-center justify-between px-space-sm py-1 rounded text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors font-label-sm text-label-sm cursor-pointer"
          type="button"
          title="Toggle rail"
        >
          <span className="flex items-center gap-space-sm">
            <Icon name="left_panel_close" className="h-4 w-4" />
            <span className="cw-sidebar-hide">Collapse rail</span>
          </span>
          <span className="font-mono text-[10px] px-1 py-0.5 bg-surface-container rounded text-outline cw-sidebar-hide">
            [
          </span>
        </button>

        <div className="flex items-center justify-between p-space-xs rounded bg-surface-container hover:bg-surface-container-high transition-colors cursor-pointer border border-surface-container-high/30">
          <div className="flex items-center gap-space-sm min-w-0">
            <div className="relative flex-shrink-0">
              <div className="w-8 h-8 rounded-full bg-[#2563eb] flex items-center justify-center text-white">
                <Icon name="person" className="h-4 w-4 text-white" />
              </div>
              <span className="absolute bottom-0 right-0 w-2 h-2 rounded-full bg-emerald-500 ring-2 ring-surface"></span>
            </div>
            <div className="min-w-0 cw-sidebar-hide">
              <p className="font-label-md text-label-md text-on-surface truncate leading-none font-semibold">
                {userName}
              </p>
              <p className="font-label-sm text-label-sm text-on-surface-variant truncate text-[10px] leading-tight mt-0.5">
                {userRole}
              </p>
            </div>
          </div>
          <Icon name="expand_more" className="h-3.5 w-3.5 text-on-surface-variant cw-sidebar-hide" />
        </div>
      </div>
    </aside>
  );
}
