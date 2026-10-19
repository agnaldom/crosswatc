export type Theme = "light" | "dark" | "system";

export const THEME_KEY = "crosswatc.theme";
export const SIDEBAR_KEY = "crosswatc.sidebar";

export function parseTheme(v: string | null): Theme {
  return v === "light" || v === "dark" ? v : "system";
}

export function isDark(theme: Theme, systemDark: boolean): boolean {
  return theme === "dark" || (theme === "system" && systemDark);
}

/** Runs in <head> before first paint so the page never flashes the wrong theme/sidebar. */
export const bootScript = `(function(){try{var d=document.documentElement;var t=localStorage.getItem("${THEME_KEY}");
var dark=t==="dark"||(t!=="light"&&matchMedia("(prefers-color-scheme: dark)").matches);d.classList.toggle("dark",dark);
if(localStorage.getItem("${SIDEBAR_KEY}")==="collapsed")d.dataset.sidebar="collapsed";}catch(e){}})();`;
