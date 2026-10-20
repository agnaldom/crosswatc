import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Crossplane Resource Views (Claims, Composites, Managed)", () => {
  test("renders Claims view with developer namespaces, claim kinds, and composite targets", async ({ app, screen }) => {
    await app.open("/infrastructure?view=claims");

    // Title
    await expect(screen.getByRole("heading", { name: "Claims" })).toBeVisible();

    // Table column headers
    await expect(screen.getByRole("columnheader", "NAME")).toBeVisible();
    await expect(screen.getByRole("columnheader", "KIND")).toBeVisible();
    await expect(screen.getByRole("columnheader", "NAMESPACE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "OWNER / SQUAD")).toBeVisible();
    await expect(screen.getByRole("columnheader", "COMPOSITE TARGET")).toBeVisible();
    await expect(screen.getByRole("columnheader", "SYNCED")).toBeVisible();
    await expect(screen.getByRole("columnheader", "READY")).toBeVisible();
    await expect(screen.getByRole("columnheader", "RESPONSIVE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "CREATED")).toBeVisible();

    // Table rows
    const table = screen.getByRole("table");
    await expect(table.getByText("payments-db-claim")).toBeVisible();
    await expect(table.getByText("redis-cache-claim")).toBeVisible();
    await expect(table.getByText("storage-bucket-claim")).toBeVisible();

    // Claim kinds
    await expect(table.getByText("PostgreSQLInstance")).toBeVisible();
    await expect(table.getByText("RedisCluster")).toBeVisible();
    await expect(table.getByText("Bucket")).toBeVisible();

    // Namespaces
    await expect(table.getByText("payments").first()).toBeVisible();
    await expect(table.getByText("platform").first()).toBeVisible();
  });

  test("renders Managed Resources view with provider, parent composite, and deletion policy", async ({ app, screen }) => {
    await app.open("/infrastructure?view=managed");

    // Title
    await expect(screen.getByRole("heading", { name: "Managed Resources" })).toBeVisible();

    // Table column headers
    await expect(screen.getByRole("columnheader", "NAME")).toBeVisible();
    await expect(screen.getByRole("columnheader", "KIND")).toBeVisible();
    await expect(screen.getByRole("columnheader", "PROVIDER")).toBeVisible();
    await expect(screen.getByRole("columnheader", "PARENT COMPOSITE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "DELETION POLICY")).toBeVisible();
    await expect(screen.getByRole("columnheader", "SYNCED")).toBeVisible();
    await expect(screen.getByRole("columnheader", "READY")).toBeVisible();
    await expect(screen.getByRole("columnheader", "RESPONSIVE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "CREATED")).toBeVisible();

    // Managed resources rows
    const table = screen.getByRole("table");
    await expect(table.getByText("payments-primary-instance")).toBeVisible();
    await expect(table.getByText("payments-firewall-rule")).toBeVisible();
    await expect(table.getByText("redis-node-pool")).toBeVisible();

    // Cloud Provider kinds
    await expect(table.getByText("FlexibleServer")).toBeVisible();
    await expect(table.getByText("FirewallRule")).toBeVisible();

    // Provider pill
    await expect(table.getByText("azure").first()).toBeVisible();

    // Parent Composite reference
    await expect(table.getByText("db-payments-prod").first()).toBeVisible();
  });

  test("renders Composite Resources view with composition and environment columns", async ({ app, screen }) => {
    await app.open("/infrastructure?view=composites");

    // Title
    await expect(screen.getByRole("heading", { name: "Composite Resources" })).toBeVisible();

    // Table column headers
    await expect(screen.getByRole("columnheader", "NAME")).toBeVisible();
    await expect(screen.getByRole("columnheader", "KIND")).toBeVisible();
    await expect(screen.getByRole("columnheader", "SYNCED")).toBeVisible();
    await expect(screen.getByRole("columnheader", "READY")).toBeVisible();
    await expect(screen.getByRole("columnheader", "RESPONSIVE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "COMPOSITION")).toBeVisible();
    await expect(screen.getByRole("columnheader", "CLAIM REF")).toBeVisible();
    await expect(screen.getByRole("columnheader", "RESOURCE REFS")).toBeVisible();
    await expect(screen.getByRole("columnheader", "CREATED")).toBeVisible();

    // Composite resources rows
    const table = screen.getByRole("table");
    await expect(table.getByText("db-payments-prod")).toBeVisible();
    await expect(table.getByText("redis-session-cache")).toBeVisible();
    await expect(table.getByText("storage-assets-bucket")).toBeVisible();

    // XR Kinds
    await expect(table.getByText("XPostgreSQLInstance")).toBeVisible();
    await expect(table.getByText("XRedisCluster")).toBeVisible();
    await expect(table.getByText("XBucket")).toBeVisible();

    // Compositions
    await expect(table.getByText("postgresql-azure-v3")).toBeVisible();
  });

  test("switches between views using the segmented tabs", async ({ app, screen, browser }) => {
    await app.open("/infrastructure");

    // Switch to Claims
    await screen.getByRole("link", "Claims (XRCs)").tap();
    await expect(browser).toHaveURL("/infrastructure?view=claims");
    await expect(screen.getByRole("heading", { name: "Claims" })).toBeVisible();

    // Switch to Managed Resources
    await screen.getByRole("link", "Managed (MRs)").tap();
    await expect(browser).toHaveURL("/infrastructure?view=managed");
    await expect(screen.getByRole("heading", { name: "Managed Resources" })).toBeVisible();

    // Switch back to Composites
    await screen.getByRole("link", "Composites (XRs)").tap();
    await expect(browser).toHaveURL("/infrastructure?view=composites");
    await expect(screen.getByRole("heading", { name: "Composite Resources" })).toBeVisible();
  });

  test("renders all 5 dropdown filters and shows 'No data found' when no results match", async ({ app, screen }) => {
    await app.open("/infrastructure?view=composites");

    // All 5 dropdown filters are present
    await expect(screen.getByRole("combobox", "Filter by Kind")).toBeVisible();
    await expect(screen.getByRole("combobox", "Filter by Synced")).toBeVisible();
    await expect(screen.getByRole("combobox", "Filter by Ready")).toBeVisible();
    await expect(screen.getByRole("combobox", "Filter by Responsive")).toBeVisible();
    await expect(screen.getByRole("combobox", "Filter by Composition")).toBeVisible();

    // Type query with no match
    const searchInput = screen.getByRole("textbox", "Search resources");
    await searchInput.fill("nonexistent-resource-xyz");

    // Verify empty state text and counter
    await expect(screen.getByText("No data found")).toBeVisible();
    await expect(screen.getByText("0 results")).toBeVisible();
  });
});
