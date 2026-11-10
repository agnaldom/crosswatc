import { ProviderInstallation } from "@/components/ProviderInstallation";

export default async function Page({ params }: { params: Promise<{ provider: string }> }) {
  const { provider } = await params;
  return <ProviderInstallation providerID={provider} />;
}
