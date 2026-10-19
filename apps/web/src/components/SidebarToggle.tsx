"use client";

import { SIDEBAR_KEY } from "@/lib/theme";

/** Collapses the sidebar to a rail (html[data-sidebar], styled in globals.css; restored by the boot script). */
export function SidebarToggle() {
  const toggle = () => {
    const d = document.documentElement;
    const collapsed = d.dataset.sidebar !== "collapsed";
    if (collapsed) d.dataset.sidebar = "collapsed";
    else delete d.dataset.sidebar;
    localStorage.setItem(SIDEBAR_KEY, collapsed ? "collapsed" : "expanded");
  };
  return (
    <button type="button" onClick={toggle} aria-label="Recolher/expandir menu" title="Recolher/expandir menu"
      className="cw-sidebar-keep mb-3 flex h-7 w-7 items-center justify-center rounded text-zinc-500 hover:bg-zinc-100 hover:text-zinc-800">
      <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.6" className="h-4 w-4" aria-hidden>
        <rect x="3" y="4" width="14" height="12" rx="2" />
        <path d="M8 4v12" />
      </svg>
    </button>
  );
}
