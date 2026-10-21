import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("TopHeader and GlobalSearch Component", () => {
  test("renders header title and search trigger with keyboard shortcut", async ({ app, screen }) => {
    await app.open("/infrastructure");

    // Wait for page to be ready
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Header title
    await expect(screen.getByRole("heading", "Crossplane Dashboard")).toBeVisible();

    // Global Search trigger button
    const searchTrigger = screen.getByRole("button", "Search resources");
    await expect(searchTrigger).toBeVisible();
    await expect(screen.getByText("⌘K")).toBeVisible();
  });

  test("opens command palette, searches resources, and displays matching results", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Open search dialog
    await screen.getByRole("button", "Search resources").tap();
    const dialog = screen.getByRole("dialog", "Search resources");
    await expect(dialog).toBeVisible();

    // Type in combobox search input
    const searchInput = screen.getByRole("combobox", "Search resources by name");
    await expect(searchInput).toBeVisible();
    await searchInput.fill("db");

    // Verify search hit appears in dialog options
    await expect(dialog.getByRole("option", /db-payments-prod/)).toBeVisible();
  });

  test("closes command palette when pressing Escape", async ({ app, screen, browser }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    await screen.getByRole("button", "Search resources").tap();
    await expect(screen.getByRole("dialog", "Search resources")).toBeVisible();

    await browser.keyboard.press("Escape");
    await expect(screen.getByRole("dialog", "Search resources")).not.toBeVisible();
  });
});
