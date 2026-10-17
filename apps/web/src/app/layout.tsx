import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Crosswatc",
  description: "Infrastructure lifecycle portal",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="pt-BR">
      <body className="min-h-screen bg-zinc-50 text-zinc-900 antialiased">{children}</body>
    </html>
  );
}
