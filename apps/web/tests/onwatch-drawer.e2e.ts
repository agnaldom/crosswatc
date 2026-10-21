import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("OnWatch Drawer Component", () => {
  test("opens and closes the slideout drawer via floating toggle button", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Floating eye trigger button
    const toggleBtn = screen.getByRole("button", "Toggle OnWatch Resources and Inspector");
    await expect(toggleBtn).toBeVisible();

    // Open drawer
    await toggleBtn.tap();
    await expect(screen.getByRole("heading", "OnWatch Resources")).toBeVisible();
    await expect(screen.getByRole("complementary", "OnWatch and Resource Inspector")).toBeVisible();

    // Close drawer
    await screen.getByRole("button", "Close OnWatch").tap();
    await expect(screen.getByRole("heading", "OnWatch Resources")).not.toBeVisible();
  });

  test("pins a resource to OnWatch, displays condition cards, and unpins it", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();
    const table = screen.getByRole("table");
    await expect(table.getByText("db-payments-prod")).toBeVisible();

    // Pin the first resource using WatchButton in the table
    const pinBtn = table.getByRole("button", "Watch db-payments-prod");
    await pinBtn.tap();

    // Open OnWatch drawer
    await screen.getByRole("button", "Toggle OnWatch Resources and Inspector").tap();
    const drawer = screen.getByRole("complementary", "OnWatch and Resource Inspector");
    await expect(drawer.getByRole("heading", "OnWatch Resources")).toBeVisible();

    // Verify pinned resource card and status conditions inside drawer
    await expect(drawer.getByText("db-payments-prod").first()).toBeVisible();
    await expect(drawer.getByText("Synced")).toBeVisible();
    await expect(drawer.getByText("Ready")).toBeVisible();
    await expect(drawer.getByRole("heading", "Status Conditions")).toBeVisible();

    // Clear all from drawer
    const clearBtn = drawer.getByRole("button", "Clear all OnWatch");
    await clearBtn.tap();

    // Verify empty state is restored
    await expect(
      drawer.getByText("No resources pinned to OnWatch. Click the eye icon on resources to monitor them here.")
    ).toBeVisible();
  });
});
