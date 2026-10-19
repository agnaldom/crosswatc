import { toneBorder } from "./ResourceNode";

const items = [
  { tone: "ok", label: "Ready / Synced" },
  { tone: "progress", label: "Criando" },
  { tone: "warn", label: "Pending / deletando" },
  { tone: "error", label: "Erro (Not Synced / Not Ready)" },
  { tone: "muted", label: "Ausente / desconhecido" },
] as const;

export function Legend() {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-zinc-600" aria-label="Legenda">
      {items.map((i) => (
        <span key={i.tone} className="inline-flex items-center gap-1.5">
          <span className="inline-block h-3 w-3 rounded-sm" style={{ border: `2px solid ${toneBorder[i.tone]}` }} />
          {i.label}
        </span>
      ))}
      <span className="inline-flex items-center gap-1.5"><span className="inline-block h-px w-5 bg-zinc-400" /> compõe</span>
      <span className="inline-flex items-center gap-1.5">
        <span className="rounded bg-zinc-900 px-1 text-[9px] text-white">XR</span> composite ·
        <span className="rounded bg-zinc-900 px-1 text-[9px] text-white">MR</span> managed resource
      </span>
    </div>
  );
}
