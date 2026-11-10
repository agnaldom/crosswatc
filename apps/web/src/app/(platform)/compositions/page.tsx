import { PageHeading } from "@/components/PageHeading";
import { CompositionsSection } from "@/components/PlatformSections";

export default function Page() {
  return <>
    <PageHeading eyebrow="Platform" title="Compositions" description="Explore how infrastructure resources are composed and delivered." />
    <CompositionsSection />
  </>;
}
