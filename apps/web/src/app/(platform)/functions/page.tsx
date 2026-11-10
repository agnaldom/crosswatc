import { PageHeading } from "@/components/PageHeading";
import { PackagesSection } from "@/components/PlatformSections";

export default function Page() {
  return <>
    <PageHeading eyebrow="Platform" title="Functions" description="Inspect the functions used by your composition pipelines." />
    <PackagesSection kind="functions" title="Functions"/>
  </>;
}
