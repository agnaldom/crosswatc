import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("TopologyView Component", () => {
  test("renders Crossplane DAG graph with nodes and legend", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Legend
    await expect(screen.getByRole("region", "Legenda")).toBeVisible();
    await expect(screen.getByText("Ready / Synced")).toBeVisible();

    // Verify Topology DAG container
    await expect(screen.getByLabel("Topologia Claim → XR → Managed Resources")).toBeVisible();

    // Verify nodes rendered in the graph
    await expect(screen.getByText("payments-db-claim")).toBeVisible();
    await expect(screen.getByText("payments-primary-instance")).toBeVisible();
    await expect(screen.getByText("payments-firewall-rule")).toBeVisible();
  });

  test("selects a topology node and displays node inspection details", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Tap on a managed resource node in the canvas
    await screen.getByText("payments-primary-instance").tap();

    // NodeDetails panel appears
    await expect(screen.getByLabel("Detalhes do recurso")).toBeVisible();
    await expect(screen.getByRole("heading", "Status Conditions")).toBeVisible();
  });
});
