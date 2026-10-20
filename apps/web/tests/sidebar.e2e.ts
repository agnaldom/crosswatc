import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Sidebar Navigation Component", () => {
  test("renders branding, context badge, and main navigation links", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Brand and Context indicator
    await expect(screen.getByText("Crosswatc")).toBeVisible();
    await expect(screen.getByText("Context")).toBeVisible();
    await expect(screen.getByText("in-cluster")).toBeVisible();

    // Main navigation links
    await expect(screen.getByRole("link", "Dashboard")).toBeVisible();
    await expect(screen.getByRole("link", "Providers")).toBeVisible();
    await expect(screen.getByRole("link", "Compositions")).toBeVisible();
    await expect(screen.getByRole("link", "XRDs")).toBeVisible();
    await expect(screen.getByRole("link", "Claims")).toBeVisible();
    await expect(screen.getByRole("link", "Managed Resources")).toBeVisible();

    // Composite Resources accordion
    const compositeBtn = screen.getByRole("button", "Composite Resources");
    await expect(compositeBtn).toBeVisible();

    // User profile footer
    await expect(screen.getByText("Logged in as")).toBeVisible();
    await expect(screen.getByText("admin")).toBeVisible();
    await expect(screen.getByRole("button", "Logout")).toBeVisible();
  });

  test("toggles Composite Resources accordion and Settings submenu", async ({ app, screen }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Click Settings to open appearance submenu
    const settingsBtn = screen.getByRole("button", "Settings");
    await settingsBtn.tap();
    await expect(screen.getByText("Appearance")).toBeVisible();

    // Toggle Composite Resources accordion closed and open
    const compositeBtn = screen.getByRole("button", "Composite Resources");
    await compositeBtn.tap();
    // Re-open
    await compositeBtn.tap();
    await expect(compositeBtn).toBeVisible();
  });

  test("navigates to Providers tab when clicking Providers link", async ({ app, screen, browser }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();
    await screen.getByRole("link", "Providers").tap();
    await expect(browser).toHaveURL("/platform?tab=Providers");
  });
});
