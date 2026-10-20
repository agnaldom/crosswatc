import type { Metadata } from "next";
import { Suspense } from "react";
import "./globals.css";
import { OnWatchDrawer } from "@/components/OnWatchDrawer";
import { Sidebar } from "@/components/Sidebar";
import { TopHeader } from "@/components/TopHeader";
import { bootScript } from "@/lib/theme";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "Crosswatc · Crossplane Dashboard",
  description: "Crossplane and GitOps lifecycle management dashboard",
  icons: {
    icon: "/icon.svg",
  },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="pt-BR" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: bootScript }} />
      </head>
      <body className="min-h-screen bg-zinc-50 text-zinc-900 antialiased dark:bg-[#090d16] dark:text-zinc-100">
        <Providers>
          <div className="flex min-h-screen">
            <Suspense fallback={<aside className="cw-sidebar w-60 shrink-0 border-r border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-[#0e1117]" />}>
              <Sidebar />
            </Suspense>
            <div className="flex min-w-0 flex-1 flex-col">
              <Suspense fallback={<header className="h-14 border-b border-zinc-200 bg-white px-6 dark:border-zinc-800 dark:bg-[#0e1117]" />}>
                <TopHeader />
              </Suspense>
              <main className="min-w-0 flex-1 px-6 py-6 sm:px-8">
                {children}
              </main>
            </div>
            <OnWatchDrawer />
          </div>
        </Providers>
      </body>
    </html>
  );
}
