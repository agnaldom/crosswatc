import { describe, expect, it } from "vitest";
import { bootScript, isDark, parseTheme } from "./theme";

describe("theme", () => {
  it("parses stored values, defaulting to system", () => {
    expect(parseTheme("dark")).toBe("dark");
    expect(parseTheme("light")).toBe("light");
    expect(parseTheme(null)).toBe("system");
    expect(parseTheme("purple")).toBe("system");
  });

  it("resolves system against the OS preference", () => {
    expect(isDark("system", true)).toBe(true);
    expect(isDark("system", false)).toBe(false);
    expect(isDark("light", true)).toBe(false);
    expect(isDark("dark", false)).toBe(true);
  });

  it("boot script is valid JS", () => {
    expect(() => new Function(bootScript)).not.toThrow();
  });
});
