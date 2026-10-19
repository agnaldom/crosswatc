import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import "./globals.css";
import { GlobalSearch } from "@/components/GlobalSearch";
import { KindNavLinks } from "@/components/KindNavLinks";
import { PlatformNavLink } from "@/components/PlatformNavLink";
import { SidebarToggle } from "@/components/SidebarToggle";
import { ThemeToggle } from "@/components/ThemeToggle";
import { WatchlistNav } from "@/components/WatchlistNav";
import { bootScript } from "@/lib/theme";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "Crosswatc",
  description: "Infrastructure lifecycle portal",
};

const nav = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/infrastructure", label: "My Infrastructure" },
  { href: "/infrastructure?view=requests", label: "Requests", disabled: true },
  { href: "/infrastructure?view=changes", label: "Changes", disabled: true },
  { href: "/infrastructure?view=operations", label: "Operations", disabled: true },
  { href: "/infrastructure?view=catalog", label: "Catalog", disabled: true },
];

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="pt-BR" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: bootScript }} />
      </head>
      <body className="min-h-screen bg-zinc-50 text-zinc-900 antialiased">
        <Providers>
          <div className="flex min-h-screen">
            <aside className="cw-sidebar w-56 shrink-0 border-r border-zinc-200 bg-white px-4 py-6">
              <SidebarToggle />
              <Link href="/infrastructure" className="block text-lg font-semibold tracking-tight">
                Crosswatc
              </Link>
              <GlobalSearch />
              <p className="mt-6 text-xs font-medium uppercase tracking-wide text-zinc-500">Infrastructure</p>
              <nav className="mt-2 flex flex-col gap-1 text-sm">
                {nav.map((n) =>
                  n.disabled ? (
                    <span key={n.label} className="rounded px-2 py-1.5 text-zinc-400" title="Em breve">
                      {n.label}
                    </span>
                  ) : (
                    <Link key={n.label} href={n.href} className="rounded px-2 py-1.5 hover:bg-zinc-100">
                      {n.label}
                    </Link>
                  ),
                )}
              </nav>
              <WatchlistNav />
              <Suspense fallback={null}>
                <KindNavLinks />
              </Suspense>
              <PlatformNavLink />
              <ThemeToggle />
            </aside>
            <main className="min-w-0 flex-1 px-8 py-6">
              {children}
            </main>
          </div>
        </Providers>
      </body>
    </html>
  );
}
