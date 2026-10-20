import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Crossplane Resource Views (Claims, Composites, Managed)", () => {
  test("renders Claims view with developer namespaces, claim kinds, and composite targets", async ({ app, screen }) => {
    await app.open("/infrastructure?view=claims");

    // Title and badge
    await expect(screen.getByRole("heading", { name: "Claims" })).toBeVisible();
    await expect(screen.getByText("Developer View (DevEx / XRC)")).toBeVisible();

    // Table column headers
    await expect(screen.getByRole("columnheader", "Name")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Kind")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Namespace")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Owner / Squad")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Composite Target")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Status")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Created")).toBeVisible();

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

    // Title and badge
    await expect(screen.getByRole("heading", { name: "Managed Resources" })).toBeVisible();
    await expect(screen.getByText("Cloud Provider Resources (MR)")).toBeVisible();

    // Table column headers
    await expect(screen.getByRole("columnheader", "Name")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Kind")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Provider")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Parent Composite")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Deletion Policy")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Status")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Created")).toBeVisible();

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

    // Title and badge
    await expect(screen.getByRole("heading", { name: "Composite Resources" })).toBeVisible();
    await expect(screen.getByText("Platform Architecture (XR)")).toBeVisible();

    // Table column headers
    await expect(screen.getByRole("columnheader", "Name")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Kind")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Composition")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Environment")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Status")).toBeVisible();
    await expect(screen.getByRole("columnheader", "Created")).toBeVisible();

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
});
