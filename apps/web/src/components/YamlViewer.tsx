"use client";

import { useMemo, useState } from "react";
import { splitMatches, tokenizeLine, type TokenKind } from "@/lib/yaml-highlight";

const color: Record<TokenKind, string> = {
  indent: "", dash: "text-zinc-400", key: "text-sky-300", colon: "text-zinc-500", string: "text-emerald-300",
  number: "text-amber-300", bool: "text-fuchsia-300", null: "text-zinc-400", comment: "text-zinc-500 italic", text: "",
};

export function YamlViewer({ yaml }: { yaml: string }) {
  const [query, setQuery] = useState("");
  const [copied, setCopied] = useState(false);
  const lines = useMemo(() => yaml.replace(/\n$/, "").split("\n"), [yaml]);
  const hits = query ? lines.filter((l) => l.toLowerCase().includes(query.toLowerCase())).length : 0;

  return (
    <div className="overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950">
      <div className="flex items-center gap-2 border-b border-zinc-800 px-3 py-2">
        <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search in YAML" aria-label="Search in YAML"
          className="w-56 rounded bg-zinc-900 px-2 py-1 text-xs text-zinc-100 placeholder:text-zinc-500" />
        {query && <span className="text-xs text-zinc-400">{hits} line(s)</span>}
        <button type="button" className="ml-auto rounded bg-zinc-800 px-2 py-1 text-xs text-zinc-100 hover:bg-zinc-700"
          onClick={async () => {
            try {
              await navigator.clipboard?.writeText(yaml);
            } catch {
              /* clipboard permissions fallback */
            }
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
          }}>
          {copied ? "Copied ✓" : "Copy"}
        </button>
      </div>
      <pre className="max-h-[560px] overflow-auto p-3 text-xs leading-5 text-zinc-100">
        <code>
          {lines.map((line, i) => {
            const hit = query && line.toLowerCase().includes(query.toLowerCase());
            return (
              <div key={i} className={hit ? "bg-yellow-500/10" : undefined}>
                <span className="mr-3 inline-block w-8 select-none text-right text-zinc-600">{i + 1}</span>
                {tokenizeLine(line).map((t, j) => (
                  <span key={j} className={color[t.kind]}>
                    {splitMatches(t.text, query).map((p, k) => p.hit
                      ? <mark key={k} className="rounded-sm bg-yellow-400 text-zinc-950">{p.text}</mark>
                      : <span key={k}>{p.text}</span>)}
                  </span>
                ))}
              </div>
            );
          })}
        </code>
      </pre>
    </div>
  );
}
