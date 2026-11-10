import { PageHeading } from "@/components/PageHeading";
import { MRAPsSection } from "@/components/PlatformSections";

export default function Page() {
  return <>
    <PageHeading eyebrow="Platform" title="Activation policies" description="Inspect which managed resource APIs your policies activate." />
    <MRAPsSection />
  </>;
}
