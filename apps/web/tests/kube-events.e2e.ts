import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("KubeEvents Component", () => {
  test("renders cluster events timeline with object filter", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Open Events tab
    await screen.getByRole("button", "Events").tap();

    // Verify filter select
    const objFilter = screen.getByRole("combobox", "Objeto");
    await expect(objFilter).toBeVisible();

    // Verify events list items
    await expect(screen.getByText("ReconcileSuccess")).toBeVisible();
    await expect(screen.getByText("Composite resource successfully synced with cloud provider")).toBeVisible();
    await expect(screen.getByText("SlowReconcile")).toBeVisible();

    // Filter by single object
    await objFilter.selectOption("FlexibleServer/payments-primary-instance");
    await expect(screen.getByText("SlowReconcile")).toBeVisible();
    await expect(screen.getByText("ReconcileSuccess")).not.toBeVisible();
  });
});
