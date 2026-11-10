import { PageHeading } from "@/components/PageHeading";
import { MRDsSection } from "@/components/PlatformSections";

export default function Page() {
  return <>
    <PageHeading eyebrow="Platform" title="Managed resource definitions" description="Inspect the managed resource APIs available in your cluster." />
    <MRDsSection />
  </>;
}
