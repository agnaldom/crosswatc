"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { PageHeading } from "@/components/PageHeading";
import { ProvidersSection } from "@/components/ProvidersSection";

export default function ProvidersPage() {
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, refetchInterval: false });
  return <>
    <PageHeading eyebrow="Platform" title="Providers" description="Manage active provider configurations and install providers from the catalog." />
    <ProvidersSection canInstall={me.data?.admin ?? false} />
  </>;
}
