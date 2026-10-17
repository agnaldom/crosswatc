"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, POLL_INTERVAL_MS } from "@/lib/api";
import { RealtimeBridge, getRealtimeStatus } from "@/lib/realtime";

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // SSE drives updates while live (spec §45); polling is the fallback when the stream is down.
            refetchInterval: () => (getRealtimeStatus() === "live" ? false : POLL_INTERVAL_MS),
            retry: (count, err) => !(err instanceof ApiError && err.status < 500) && count < 2,
          },
        },
      }),
  );
  return (
    <QueryClientProvider client={client}>
      <RealtimeBridge />
      {children}
    </QueryClientProvider>
  );
}
