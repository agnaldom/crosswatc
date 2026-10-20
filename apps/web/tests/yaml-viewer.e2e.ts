import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("YamlViewer Component", () => {
  test("renders syntax-highlighted YAML manifest with search and copy controls", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Open YAML tab
    await screen.getByRole("button", "YAML").tap();

    // Verify code lines
    await expect(screen.getByText("apiVersion")).toBeVisible();
    await expect(screen.getByText("storageGB")).toBeVisible();

    // Search inside YAML
    const searchInput = screen.getByRole("textbox", "Buscar no YAML");
    await expect(searchInput).toBeVisible();
    await searchInput.fill("storageGB");

    // Verify search hit count
    await expect(screen.getByText("1 linha(s)")).toBeVisible();

    // Copy button
    const copyBtn = screen.getByRole("button", "Copiar");
    await expect(copyBtn).toBeVisible();
    await copyBtn.tap();
    await expect(screen.getByText("Copiado ✓")).toBeVisible();
  });
});
