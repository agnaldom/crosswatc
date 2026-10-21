import type { Metadata } from "next";
import { Suspense } from "react";
import "./globals.css";
import { OnWatchDrawer } from "@/components/OnWatchDrawer";
import { Sidebar } from "@/components/Sidebar";
import { TopHeader } from "@/components/TopHeader";
import { bootScript } from "@/lib/theme";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "Crosswatc · Infrastructure Control Plane",
  description: "Crossplane and GitOps lifecycle management control plane",
  icons: {
    icon: "/icon.svg",
  },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="" />
        <link
          href="https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600&family=Inter:wght@400;500;600&display=swap"
          rel="stylesheet"
        />
        <script dangerouslySetInnerHTML={{ __html: bootScript }} />
      </head>
      <body className="bg-surface text-on-surface font-body-md antialiased min-h-screen transition-colors duration-150">
        <Providers>
          <div className="min-h-screen bg-surface">
            <Suspense fallback={<aside className="fixed left-0 top-0 h-screen w-60 bg-surface-container-low" />}>
              <Sidebar />
            </Suspense>
            <div className="pl-60 transition-[padding] duration-200 cw-main-container min-h-screen flex flex-col">
              <Suspense fallback={<header className="fixed top-0 left-60 right-0 h-14 bg-surface/80" />}>
                <TopHeader />
              </Suspense>
              <main className="w-full pt-14 bg-surface min-h-screen flex-1">
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
