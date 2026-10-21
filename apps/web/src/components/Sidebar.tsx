"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { useState } from "react";
import { Logo } from "@/components/Logo";
import { api } from "@/lib/api";
import { SIDEBAR_KEY } from "@/lib/theme";
import { useTheme } from "@/lib/use-theme";

export function Sidebar() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const currentKind = searchParams.get("kind");
  const currentTab = searchParams.get("tab");
  const currentView = searchParams.get("view");

  const [compositeOpen, setCompositeOpen] = useState(true);
  const [settingsOpen, setSettingsOpen] = useState(false);

  const kindsQuery = useQuery({
    queryKey: ["resource-kinds"],
    queryFn: api.resourceKinds,
    staleTime: 60_000,
  });

  const meQuery = useQuery({
    queryKey: ["me"],
    queryFn: api.me,
    staleTime: 60_000,
  });

  const { theme, toggle: toggleTheme } = useTheme();

  const toggleSidebar = () => {
    const d = document.documentElement;
    const collapsed = d.dataset.sidebar !== "collapsed";
    if (collapsed) d.dataset.sidebar = "collapsed";
    else delete d.dataset.sidebar;
    localStorage.setItem(SIDEBAR_KEY, collapsed ? "collapsed" : "expanded");
  };

  const kinds = kindsQuery.data?.items ?? [];
  const userName = meQuery.data?.user || "admin";

  const isPlatformTab = (tabName: string) => pathname === "/platform" && currentTab === tabName;

  return (
    <aside className="cw-sidebar flex w-60 shrink-0 flex-col justify-between border-r border-zinc-200 bg-white px-3.5 py-4 transition-all duration-200 dark:border-zinc-800 dark:bg-[#0e1117]">
      {/* Top Section */}
      <div className="space-y-4">
        {/* Logo and Collapse Toggle */}
        <div className="flex items-center justify-between">
          <Link href="/infrastructure" className="block py-1">
            <Logo size="md" showTagline={false} />
          </Link>
          <button
            type="button"
            onClick={toggleSidebar}
            title="Recolher barra lateral"
            className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
          >
            <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
              <path d="M12 15l-5-5 5-5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </button>
        </div>

        {/* CONTEXT Section */}
        <div>
          <div className="px-1 text-[10px] font-bold uppercase tracking-wider text-zinc-400 dark:text-zinc-500">
            Context
          </div>
          <div className="mt-1.5 flex items-center justify-between rounded-lg border border-zinc-200 bg-zinc-50 px-2.5 py-1.5 text-xs font-medium text-zinc-700 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300">
            <div className="flex items-center gap-1.5 truncate">
              <span className="h-2 w-2 rounded-full bg-emerald-500" />
              <span className="truncate">in-cluster</span>
            </div>
            <svg viewBox="0 0 20 20" fill="currentColor" className="h-3.5 w-3.5 text-zinc-400">
              <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clipRule="evenodd" />
            </svg>
          </div>
        </div>

        {/* MENU Section */}
        <div>
          <div className="px-1 text-[10px] font-bold uppercase tracking-wider text-zinc-400 dark:text-zinc-500">
            Menu
          </div>
          <nav className="mt-1.5 flex flex-col gap-0.5 text-xs font-medium text-zinc-600 dark:text-zinc-400">
            {/* Dashboard */}
            <Link
              href="/dashboard"
              className={`flex items-center gap-2 rounded-lg px-2.5 py-2 transition ${
                pathname === "/dashboard"
                  ? "bg-zinc-100 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-white"
                  : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              }`}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                <rect x="3" y="3" width="7" height="7" rx="1" />
                <rect x="14" y="3" width="7" height="7" rx="1" />
                <rect x="14" y="14" width="7" height="7" rx="1" />
                <rect x="3" y="14" width="7" height="7" rx="1" />
              </svg>
              <span>Dashboard</span>
            </Link>

            {/* Providers */}
            <Link
              href="/platform?tab=Providers"
              className={`flex items-center gap-2 rounded-lg px-2.5 py-2 transition ${
                isPlatformTab("Providers")
                  ? "bg-zinc-100 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-white"
                  : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              }`}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
                <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
                <line x1="12" y1="22.08" x2="12" y2="12" />
              </svg>
              <span>Providers</span>
            </Link>

            {/* Compositions */}
            <Link
              href="/platform?tab=Compositions"
              className={`flex items-center gap-2 rounded-lg px-2.5 py-2 transition ${
                isPlatformTab("Compositions")
                  ? "bg-zinc-100 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-white"
                  : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              }`}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                <polygon points="12 2 2 7 12 12 22 7 12 2" />
                <polyline points="2 17 12 22 22 17" />
                <polyline points="2 12 12 17 22 12" />
              </svg>
              <span>Compositions</span>
            </Link>

            {/* XRDs */}
            <Link
              href="/platform?tab=XRDs"
              className={`flex items-center gap-2 rounded-lg px-2.5 py-2 transition ${
                isPlatformTab("XRDs")
                  ? "bg-zinc-100 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-white"
                  : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              }`}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                <polyline points="14 2 14 8 20 8" />
                <line x1="16" y1="13" x2="8" y2="13" />
                <line x1="16" y1="17" x2="8" y2="17" />
              </svg>
              <span>XRDs</span>
            </Link>

            {/* Composite Resources (Collapsible Accordion) */}
            <div>
              <button
                type="button"
                onClick={() => setCompositeOpen((prev) => !prev)}
                className={`flex w-full items-center justify-between rounded-lg px-2.5 py-2 transition ${
                  pathname === "/infrastructure" && (!currentView || currentView === "composites")
                    ? "font-semibold text-zinc-900 dark:text-white"
                    : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
                }`}
              >
                <div className="flex items-center gap-2">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                    <rect x="2" y="7" width="20" height="14" rx="2" ry="2" />
                    <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
                  </svg>
                  <span>Composite Resources</span>
                </div>
                <svg
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  className={`h-4 w-4 text-zinc-400 transition-transform ${compositeOpen ? "rotate-180" : ""}`}
                >
                  <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clipRule="evenodd" />
                </svg>
              </button>

              {/* Submenu Kinds List */}
              {compositeOpen && (
                <div className="ml-5 mt-1 flex flex-col gap-0.5 border-l border-zinc-200 pl-2 dark:border-zinc-800">
                  <Link
                    href="/infrastructure?view=composites"
                    className={`flex items-center justify-between rounded-md px-2 py-1.5 text-xs transition ${
                      pathname === "/infrastructure" && (!currentView || currentView === "composites") && !currentKind
                        ? "bg-zinc-200/70 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-cyan-400"
                        : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/50 dark:hover:text-zinc-200"
                    }`}
                  >
                    <span>All Composites</span>
                  </Link>
                  {kinds.length > 0 ? (
                    kinds.map((k) => {
                      const isActive = pathname === "/infrastructure" && currentKind === k.kind;
                      return (
                        <Link
                          key={k.kind}
                          href={`/infrastructure?view=composites&kind=${encodeURIComponent(k.kind)}`}
                          className={`flex items-center justify-between rounded-md px-2 py-1.5 text-xs transition ${
                            isActive
                              ? "bg-zinc-200/70 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-cyan-400"
                              : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/50 dark:hover:text-zinc-200"
                          }`}
                        >
                          <span className="truncate">{k.kind}</span>
                          <span className="text-[10px] text-zinc-400 tabular-nums">{k.count}</span>
                        </Link>
                      );
                    })
                  ) : (
                    <span className="px-2 py-1 text-[11px] text-zinc-400">No kinds</span>
                  )}
                </div>
              )}
            </div>

            {/* Claims */}
            <Link
              href="/infrastructure?view=claims"
              className={`flex items-center gap-2 rounded-lg px-2.5 py-2 transition ${
                pathname === "/infrastructure" && currentView === "claims"
                  ? "bg-zinc-100 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-white"
                  : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              }`}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
                <rect x="8" y="2" width="8" height="4" rx="1" ry="1" />
              </svg>
              <span>Claims</span>
            </Link>

            {/* Managed Resources */}
            <Link
              href="/infrastructure?view=managed"
              className={`flex items-center gap-2 rounded-lg px-2.5 py-2 transition ${
                pathname === "/infrastructure" && currentView === "managed"
                  ? "bg-zinc-100 font-semibold text-zinc-900 dark:bg-zinc-800 dark:text-white"
                  : "hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              }`}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                <ellipse cx="12" cy="5" rx="9" ry="3" />
                <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
                <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
              </svg>
              <span>Managed Resources</span>
            </Link>

            {/* Settings */}
            <div>
              <button
                type="button"
                onClick={() => setSettingsOpen((prev) => !prev)}
                className="flex w-full items-center justify-between rounded-lg px-2.5 py-2 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800/60 dark:hover:text-zinc-200"
              >
                <div className="flex items-center gap-2">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="h-4 w-4">
                    <circle cx="12" cy="12" r="3" />
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
                  </svg>
                  <span>Settings</span>
                </div>
                <svg
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  className={`h-4 w-4 text-zinc-400 transition-transform ${settingsOpen ? "rotate-180" : ""}`}
                >
                  <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clipRule="evenodd" />
                </svg>
              </button>

              {settingsOpen && (
                <div className="ml-5 mt-1 flex flex-col gap-0.5 border-l border-zinc-200 pl-2 dark:border-zinc-800">
                  <button
                    type="button"
                    onClick={toggleTheme}
                    aria-label="Toggle appearance theme"
                    className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-xs text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
                  >
                    <span>Appearance</span>
                    <span className="capitalize text-[10px] text-zinc-400">{theme}</span>
                  </button>
                  <span className="rounded-md px-2 py-1.5 text-xs text-zinc-400">User Management</span>
                  <span className="rounded-md px-2 py-1.5 text-xs text-zinc-400">Contexts</span>
                </div>
              )}
            </div>
          </nav>
        </div>
      </div>

      {/* Bottom Section - User Profile Footer */}
      <div className="border-t border-zinc-200 pt-3 dark:border-zinc-800">
        <div className="rounded-lg border border-zinc-200/80 bg-zinc-50 p-2.5 dark:border-zinc-800/80 dark:bg-zinc-900/40">
          <div className="text-[10px] text-zinc-400 dark:text-zinc-500">Logged in as</div>
          <div className="truncate text-xs font-bold text-zinc-900 dark:text-zinc-100">{userName}</div>
          <button
            type="button"
            onClick={() => alert(`Session for ${userName}. To switch, configure authentication headers.`)}
            className="mt-2 flex items-center gap-1.5 text-xs font-medium text-rose-600 hover:underline dark:text-rose-400"
          >
            <svg viewBox="0 0 20 20" fill="currentColor" className="h-3.5 w-3.5">
              <path fillRule="evenodd" d="M3 3a1 1 0 00-1 1v12a1 1 0 102 0V4a1 1 0 00-1-1zm10.293 9.293a1 1 0 001.414 1.414l3-3a1 1 0 000-1.414l-3-3a1 1 0 10-1.414 1.414L14.586 9H7a1 1 0 100 2h7.586l-1.293 1.293z" clipRule="evenodd" />
            </svg>
            <span>Logout</span>
          </button>
        </div>
      </div>
    </aside>
  );
}
