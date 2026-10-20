import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Platform Catalog Component (Providers, Functions, XRDs, Compositions)", () => {
  test("renders Providers tab with search, status filters, and cards view", async ({ app, screen }) => {
    await app.open("/platform");

    // Title and tabs
    await expect(screen.getByRole("heading", { name: "Platform Catalog" })).toBeVisible();
    await expect(screen.getByRole("tab", { name: "Providers" })).toBeVisible();
    await expect(screen.getByRole("tab", { name: "Functions" })).toBeVisible();
    await expect(screen.getByRole("tab", { name: "XRDs" })).toBeVisible();
    await expect(screen.getByRole("tab", { name: "Compositions" })).toBeVisible();

    // Default tab is Providers
    await expect(screen.getByText("provider-azure-db")).toBeVisible();
    await expect(screen.getByText("provider-kubernetes")).toBeVisible();

    // Counter
    await expect(screen.getByText("2 results")).toBeVisible();

    // Status filter
    const statusSelect = screen.getByRole("combobox", "Filter by Status");
    await expect(statusSelect).toBeVisible();

    // Filter by unhealthy using selectOption
    await statusSelect.selectOption("Unhealthy");
    await expect(screen.getByText("provider-kubernetes")).toBeVisible();
    await expect(screen.getByText("Failed to connect to cluster API")).toBeVisible();
    await expect(screen.getByText("1 results")).toBeVisible();

    // Reset status filter
    await statusSelect.selectOption("All Status");
    await expect(screen.getByText("2 results")).toBeVisible();

    // Search filter
    const searchInput = screen.getByRole("textbox", "Search Providers");
    await searchInput.fill("azure");
    await expect(screen.getByText("provider-azure-db")).toBeVisible();
    await expect(screen.getByText("1 results")).toBeVisible();

    // Nonexistent search -> No data found
    await searchInput.fill("nonexistent-provider-xyz");
    await expect(screen.getByText("No data found")).toBeVisible();
    await expect(screen.getByText("0 results")).toBeVisible();

    // Clear search
    await searchInput.fill("");
    await expect(screen.getByText("2 results")).toBeVisible();

    // Toggle to table view
    await screen.getByRole("button", "Table view").tap();
    await expect(screen.getByRole("columnheader", "NAME")).toBeVisible();
    await expect(screen.getByRole("columnheader", "HEALTH")).toBeVisible();
    await expect(screen.getByRole("columnheader", "INSTALLED")).toBeVisible();
    await expect(screen.getByRole("columnheader", "PACKAGE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "REVISION")).toBeVisible();
  });

  test("switches to Functions tab and filters functions", async ({ app, screen }) => {
    await app.open("/platform");

    // Switch to Functions tab
    await screen.getByRole("tab", { name: "Functions" }).tap();

    await expect(screen.getByText("function-patch-and-transform")).toBeVisible();
    await expect(screen.getByText("function-auto-ready")).toBeVisible();
    await expect(screen.getByText("2 results")).toBeVisible();

    // Search function
    const searchInput = screen.getByRole("textbox", "Search Functions");
    await searchInput.fill("patch");
    await expect(screen.getByText("function-patch-and-transform")).toBeVisible();
    await expect(screen.getByText("1 results")).toBeVisible();
  });

  test("switches to XRDs tab and displays Established & Offered statuses", async ({ app, screen }) => {
    await app.open("/platform");

    // Switch to XRDs tab
    await screen.getByRole("tab", { name: "XRDs" }).tap();

    // Table columns
    await expect(screen.getByRole("columnheader", "NAME")).toBeVisible();
    await expect(screen.getByRole("columnheader", "GROUP")).toBeVisible();
    await expect(screen.getByRole("columnheader", "RESOURCE KIND")).toBeVisible();
    await expect(screen.getByRole("columnheader", "CLAIM KIND")).toBeVisible();
    await expect(screen.getByRole("columnheader", "STATUS")).toBeVisible();

    // Data rows
    await expect(screen.getByText("xpostgresqlinstances.database.example.org")).toBeVisible();

    // Dual status badges
    await expect(screen.getByText("Established").first()).toBeVisible();
    await expect(screen.getByText("Offered").first()).toBeVisible();

    // Group filter
    const groupSelect = screen.getByRole("combobox", "Filter by Group");
    await groupSelect.selectOption("database.example.org");
    await expect(screen.getByText("xpostgresqlinstances.database.example.org")).toBeVisible();
    await expect(screen.getByText("1 results")).toBeVisible();
  });

  test("switches to Compositions tab and displays pipeline functions", async ({ app, screen }) => {
    await app.open("/platform");

    // Switch to Compositions tab
    await screen.getByRole("tab", { name: "Compositions" }).tap();

    await expect(screen.getByRole("columnheader", "NAME")).toBeVisible();
    await expect(screen.getByRole("columnheader", "XR KIND")).toBeVisible();
    await expect(screen.getByRole("columnheader", "MODE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "STEPS")).toBeVisible();
    await expect(screen.getByRole("columnheader", "FUNCTIONS")).toBeVisible();

    // Compositions row
    await expect(screen.getByText("postgresql-azure-v3")).toBeVisible();

    // Pipeline function pill
    await expect(screen.getByText("function-patch-and-transform").first()).toBeVisible();
  });
});
