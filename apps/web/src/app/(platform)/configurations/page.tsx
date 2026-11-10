import { PageHeading } from "@/components/PageHeading";
import { PackagesSection } from "@/components/PlatformSections";

export default function Page() {
  return <>
    <PageHeading eyebrow="Platform" title="Configurations" description="Inspect the configuration packages installed in your control plane." />
    <PackagesSection kind="configurations" title="Configurations"/>
  </>;
}
