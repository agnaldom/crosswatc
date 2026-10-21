import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("KubeEvents Component", () => {
  test("renders cluster events timeline with object filter, severity filter, and search", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Open Events tab
    await screen.getByRole("button", "Events").tap();

    // Verify filter select
    const objFilter = screen.getByRole("combobox", "Object");
    await expect(objFilter).toBeVisible();

    // Verify events list items
    await expect(screen.getByText("ReconcileSuccess")).toBeVisible();
    await expect(screen.getByText("Composite resource successfully synced with cloud provider")).toBeVisible();
    await expect(screen.getByText("SlowReconcile")).toBeVisible();

    // Verify stats badges
    await expect(screen.getByText("Total: 2")).toBeVisible();
    await expect(screen.getByText("1 Warnings")).toBeVisible();
    await expect(screen.getByText("1 Normal")).toBeVisible();

    // Filter by Severity: Warning Only
    const typeFilter = screen.getByRole("combobox", "Filter by Type");
    await typeFilter.selectOption("Warning Only");
    await expect(screen.getByText("SlowReconcile")).toBeVisible();
    await expect(screen.getByText("ReconcileSuccess")).not.toBeVisible();

    // Reset severity filter
    await typeFilter.selectOption("All Types");
    await expect(screen.getByText("ReconcileSuccess")).toBeVisible();

    // Search events by message keyword
    const searchInput = screen.getByRole("textbox", "Search events");
    await searchInput.fill("cloud provider");
    await expect(screen.getByText("ReconcileSuccess")).toBeVisible();
    await expect(screen.getByText("SlowReconcile")).not.toBeVisible();

    // Clear search
    await searchInput.fill("");
    await expect(screen.getByText("SlowReconcile")).toBeVisible();

    // Filter by single object
    await objFilter.selectOption("FlexibleServer/payments-primary-instance");
    await expect(screen.getByText("SlowReconcile")).toBeVisible();
    await expect(screen.getByText("ReconcileSuccess")).not.toBeVisible();
  });
});
