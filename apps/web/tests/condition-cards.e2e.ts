import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("ConditionCards Component", () => {
  test("displays Crossplane condition status cards with reasons and messages", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Click on Status tab in the Detail pane
    await screen.getByRole("button", "Status").tap();

    const detailPane = screen.getByLabel("Selected resource details");

    // Verify condition cards inside detail pane
    await expect(detailPane.getByText(/WatchCircuitClosed/)).toBeVisible();
    await expect(detailPane.getByText(/ReconcileSuccess/)).toBeVisible();
    await expect(detailPane.getByText(/Available/)).toBeVisible();
  });
});
