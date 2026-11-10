import { PageHeading } from "@/components/PageHeading";
import { XRDsSection } from "@/components/PlatformSections";

export default function Page() {
  return <>
    <PageHeading eyebrow="Platform" title="Resource definitions" description="Explore composite resource definitions and the APIs they expose." />
    <XRDsSection />
  </>;
}
