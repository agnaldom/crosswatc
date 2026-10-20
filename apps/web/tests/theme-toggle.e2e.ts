import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Theme Toggle Component", () => {
  test("toggles theme between dark and light modes", async ({ app, screen, browser }) => {
    await app.open("/infrastructure");
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();

    // Open Settings submenu
    await screen.getByRole("button", "Settings").tap();
    await expect(screen.getByText("Appearance")).toBeVisible();

    // Tap Appearance to toggle theme
    await screen.getByRole("button", "Alternar tema de aparência").tap();

    // Verify localStorage has theme key set
    const themeInStorage = await browser.evaluate(() => localStorage.getItem("crosswatc.theme"));
    expect(["dark", "light"]).toContain(themeInStorage);
  });
});
