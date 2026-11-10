import { ErrorState } from "@/components/States";

export function LoadMore({ query }: { query: { hasNextPage: boolean; isFetchingNextPage: boolean; isFetchNextPageError: boolean; error: Error | null; fetchNextPage: () => unknown } }) {
  return <div className="space-y-3">
    {query.isFetchNextPageError && <ErrorState error={query.error} />}
    {query.hasNextPage && <button type="button" className="cw-button" disabled={query.isFetchingNextPage} onClick={() => query.fetchNextPage()}>{query.isFetchingNextPage ? "Loading…" : "Load more"}</button>}
  </div>;
}
