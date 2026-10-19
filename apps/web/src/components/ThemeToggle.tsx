"use client";

import { useEffect, useState } from "react";
import { isDark, parseTheme, THEME_KEY, type Theme } from "@/lib/theme";

const options: { value: Theme; label: string }[] = [
  { value: "light", label: "Claro" },
  { value: "dark", label: "Escuro" },
  { value: "system", label: "Sistema" },
];

/** Light/dark/system selector persisted in localStorage; follows OS changes while on "system". */
export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>("system");
  useEffect(() => setTheme(parseTheme(localStorage.getItem(THEME_KEY))), []); // eslint-disable-line react-hooks/set-state-in-effect -- read after hydration
  useEffect(() => {
    const mq = matchMedia("(prefers-color-scheme: dark)");
    const apply = () => document.documentElement.classList.toggle("dark", isDark(theme, mq.matches));
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, [theme]);
  return (
    <label className="mt-6 block text-xs text-zinc-500">
      Tema
      <select value={theme} aria-label="Tema"
        onChange={(e) => {
          const t = parseTheme(e.target.value);
          localStorage.setItem(THEME_KEY, t);
          setTheme(t);
        }}
        className="mt-1 block w-full rounded-md border border-zinc-300 bg-white px-2 py-1 text-sm text-zinc-700">
        {options.map((o) => (
          <option key={o.value} value={o.value}>{o.label}</option>
        ))}
      </select>
    </label>
  );
}
