/** Canonical pages shared by the sidebar, breadcrumb and legacy redirects. */
export const platformPages = [
  { label: "Providers", href: "/providers", icon: "hub", legacyTab: "Providers" },
  { label: "Functions", href: "/functions", icon: "code_blocks", legacyTab: "Functions" },
  { label: "Configurations", href: "/configurations", icon: "tune", legacyTab: "Configurations" },
  { label: "Compositions", href: "/compositions", icon: "schema", legacyTab: "Compositions" },
  { label: "Resource definitions", href: "/resource-definitions", icon: "account_tree", legacyTab: "XRDs" },
  { label: "Managed resource definitions", href: "/managed-resource-definitions", icon: "layers", legacyTab: "MRDs" },
  { label: "Activation policies", href: "/activation-policies", icon: "tune", legacyTab: "Activation Policies" },
] as const;
