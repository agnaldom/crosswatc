import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Infrastructure Master-Detail Component", () => {
  test("renders resources table with multi-status pills", async ({ app, screen }) => {
    await app.open("/infrastructure");

    // Wait for table to load
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();
    const table = screen.getByRole("table");
    await expect(table.getByText("db-payments-prod")).toBeVisible();

    // Table headings matching user UI design
    await expect(screen.getByRole("columnheader", "NAME")).toBeVisible();
    await expect(screen.getByRole("columnheader", "KIND")).toBeVisible();
    await expect(screen.getByRole("columnheader", "SYNCED")).toBeVisible();
    await expect(screen.getByRole("columnheader", "READY")).toBeVisible();
    await expect(screen.getByRole("columnheader", "RESPONSIVE")).toBeVisible();
    await expect(screen.getByRole("columnheader", "COMPOSITION")).toBeVisible();
    await expect(screen.getByRole("columnheader", "CLAIM REF")).toBeVisible();
    await expect(screen.getByRole("columnheader", "RESOURCE REFS")).toBeVisible();
    await expect(screen.getByRole("columnheader", "CREATED")).toBeVisible();

    // Multi-status pills
    await expect(table.getByText("Synced").first()).toBeVisible();
    await expect(table.getByText("Ready").first()).toBeVisible();
    await expect(table.getByText("Responsive").first()).toBeVisible();

    // Resource rows
    await expect(table.getByText("redis-session-cache")).toBeVisible();
    await expect(table.getByText("storage-assets-bucket")).toBeVisible();
  });

  test("filters resources using the search box", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();
    const table = screen.getByRole("table");
    await expect(table.getByText("redis-session-cache")).toBeVisible();

    const searchInput = screen.getByRole("textbox", "Search resources");
    await expect(searchInput).toBeVisible();

    // Type query to filter
    await searchInput.fill("redis");
    await expect(table.getByText("redis-session-cache")).toBeVisible();
    await expect(table.getByText("storage-assets-bucket")).not.toBeVisible();

    // Clear search
    await searchInput.fill("");
    await expect(table.getByText("storage-assets-bucket")).toBeVisible();
  });

  test("filters resources using the status dropdown", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();
    const table = screen.getByRole("table");
    await expect(table.getByText("redis-session-cache")).toBeVisible();

    const statusDropdown = screen.getByRole("combobox", "Filter by Synced");
    await expect(statusDropdown).toBeVisible();

    // Select synced filter using label
    await statusDropdown.selectOption("Synced");
    await expect(table.getByText("redis-session-cache")).toBeVisible();
  });

  test("selects a resource row and displays the detail pane with tabs", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();
    const table = screen.getByRole("table");
    await expect(table.getByText("redis-session-cache")).toBeVisible();

    // Click on resource row in table
    await table.getByText("redis-session-cache").tap();

    // Detail Pane header displays the selected resource
    const detailPane = screen.getByLabel("Detalhes do recurso selecionado");
    await expect(detailPane.getByText("XRedisCluster")).toBeVisible();

    // Detail pane tabs
    await expect(screen.getByRole("button", "Relations")).toBeVisible();
    await expect(screen.getByRole("button", "YAML")).toBeVisible();
    await expect(screen.getByRole("button", "Status")).toBeVisible();
    await expect(screen.getByRole("button", "Events")).toBeVisible();
    await expect(screen.getByRole("button", "Overview")).toBeVisible();

    // Switch to YAML tab
    await screen.getByRole("button", "YAML").tap();
    await expect(screen.getByText("apiVersion")).toBeVisible();

    // Switch to Status tab
    await screen.getByRole("button", "Status").tap();
    await expect(detailPane.getByText(/WatchCircuitClosed/)).toBeVisible();

    // Switch to Events tab
    await screen.getByRole("button", "Events").tap();
    await expect(screen.getByRole("combobox", "Objeto")).toBeVisible();

    // Switch to Overview tab
    await screen.getByRole("button", "Overview").tap();
    await expect(screen.getByText("Metadados")).toBeVisible();
  });
});
