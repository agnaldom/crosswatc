// Minimal YAML line tokenizer for read-only display (no dependency, no HTML injection: callers render tokens as text).

export type TokenKind = "indent" | "dash" | "key" | "colon" | "string" | "number" | "bool" | "null" | "comment" | "text";
export interface Token {
  kind: TokenKind;
  text: string;
}

const scalar = (v: string): TokenKind => {
  const t = v.trim();
  if (t === "") return "text";
  if (/^(true|false)$/i.test(t)) return "bool";
  if (/^(null|~)$/i.test(t)) return "null";
  if (/^-?\d+(\.\d+)?([eE][+-]?\d+)?$/.test(t)) return "number";
  if (/^["'].*["']$/.test(t)) return "string";
  return "string";
};

export function tokenizeLine(line: string): Token[] {
  const out: Token[] = [];
  const m = /^(\s*)(.*)$/.exec(line)!;
  if (m[1]) out.push({ kind: "indent", text: m[1] });
  let rest = m[2];
  if (rest.startsWith("#")) return [...out, { kind: "comment", text: rest }];
  if (rest.startsWith("- ") || rest === "-") {
    out.push({ kind: "dash", text: rest.startsWith("- ") ? "- " : "-" });
    rest = rest.slice(2);
  }
  const kv = /^((?:"[^"]*"|'[^']*'|[^:#'"][^:#]*?))(:)(\s+|$)(.*)$/.exec(rest);
  if (kv) {
    out.push({ kind: "key", text: kv[1] }, { kind: "colon", text: kv[2] + kv[3] });
    rest = kv[4];
  }
  if (rest) {
    const c = rest.indexOf(" #");
    const value = c >= 0 ? rest.slice(0, c) : rest;
    if (value) out.push({ kind: scalar(value), text: value });
    if (c >= 0) out.push({ kind: "comment", text: rest.slice(c) });
  }
  return out;
}

/** Splits text into [before, match, after…] chunks for case-insensitive search highlighting. */
export function splitMatches(text: string, query: string): { text: string; hit: boolean }[] {
  if (!query) return [{ text, hit: false }];
  const out: { text: string; hit: boolean }[] = [];
  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  let i = 0;
  for (let j = lower.indexOf(q); j >= 0; j = lower.indexOf(q, i)) {
    if (j > i) out.push({ text: text.slice(i, j), hit: false });
    out.push({ text: text.slice(j, j + q.length), hit: true });
    i = j + q.length;
  }
  if (i < text.length) out.push({ text: text.slice(i), hit: false });
  return out;
}
