"use client";

import { usePathname, useSearchParams } from "next/navigation";
import { GlobalSearch } from "@/components/GlobalSearch";

export function TopHeader() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const kind = searchParams.get("kind");

  let pageTitle = "Crossplane Dashboard";
  if (pathname === "/dashboard") {
    pageTitle = "Overview Dashboard";
  } else if (pathname === "/platform") {
    pageTitle = "Platform · Crossplane Catalog";
  } else if (kind) {
    pageTitle = kind;
  }

  return (
    <header className="flex h-14 items-center justify-between border-b border-zinc-200 bg-white px-6 dark:border-zinc-800 dark:bg-[#0e1117]">
      <h1 className="text-base font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">
        {pageTitle}
      </h1>

      <div className="flex items-center gap-3">
        {/* Global Search Trigger */}
        <GlobalSearch />
      </div>
    </header>
  );
}
