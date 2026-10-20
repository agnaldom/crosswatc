"use client";

import { useEffect, useState } from "react";
import { parseTheme, THEME_KEY, type Theme } from "./theme";

export function useTheme() {
  const [theme, setTheme] = useState<Theme>("system");
  useEffect(() => {
    setTheme(parseTheme(localStorage.getItem(THEME_KEY))); // eslint-disable-line react-hooks/set-state-in-effect -- read after hydration
  }, []);
  const toggle = () => {
    const next: Theme = theme === "dark" ? "light" : "dark";
    localStorage.setItem(THEME_KEY, next);
    setTheme(next);
    document.documentElement.classList.toggle("dark", next === "dark");
  };
  return { theme, setTheme, toggle };
}
