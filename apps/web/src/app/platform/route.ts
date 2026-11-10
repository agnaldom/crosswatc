import { platformPages } from "@/lib/platform-navigation";

/** Redirect old bookmarks before rendering the workspace or checking its session. */
export function GET(request: Request) {
  const tabs = new URL(request.url).searchParams.getAll("tab");
  const page = tabs.length === 1 ? platformPages.find((item) => item.legacyTab === tabs[0]) : undefined;
  return new Response(null, { status: 307, headers: { Location: page?.href ?? "/providers", "Cache-Control": "no-store" } });
}
