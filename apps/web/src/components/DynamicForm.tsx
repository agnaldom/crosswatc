"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMemo } from "react";
import { useForm, type FieldValues } from "react-hook-form";
import { buildSchema } from "@/lib/form-schema";
import type { FormField } from "@/lib/types";

const inputClass = "w-full rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm disabled:bg-zinc-100 disabled:text-zinc-500";

export function DynamicForm({ fields, values, submitLabel, onSubmit, busy }: {
  fields: FormField[];
  values: Record<string, unknown>;
  submitLabel: string;
  onSubmit: (v: Record<string, unknown>) => void;
  busy?: boolean;
}) {
  const schema = useMemo(() => buildSchema(fields), [fields]);
  const defaults = useMemo(() => {
    const d: Record<string, unknown> = {};
    for (const f of fields) d[f.key] = values[f.key] ?? f.default ?? (f.type === "boolean" ? false : "");
    return d;
  }, [fields, values]);
  const { register, handleSubmit, formState: { errors } } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: defaults,
  });

  return (
    <form onSubmit={handleSubmit((v) => onSubmit(v))} className="space-y-4" noValidate>
      {fields.map((f) => {
        const id = `field-${f.key}`;
        const err = errors[f.key]?.message as string | undefined;
        return (
          <div key={f.key} className="grid gap-1 md:grid-cols-[14rem_1fr] md:items-start">
            <label htmlFor={id} className="pt-1.5 text-sm font-medium">
              {f.title}
              {f.required && <span className="text-rose-600"> *</span>}
              {!f.editable && <span className="ml-1 text-xs font-normal text-zinc-500">(somente leitura)</span>}
            </label>
            <div>
              {f.type === "boolean" ? (
                <input id={id} type="checkbox" disabled={!f.editable} {...register(f.key)} />
              ) : f.enum?.length ? (
                <select id={id} disabled={!f.editable} className={inputClass} {...register(f.key)}>
                  {f.enum.map((o) => <option key={String(o)} value={String(o)}>{String(o)}</option>)}
                </select>
              ) : (
                <input id={id} disabled={!f.editable} className={inputClass}
                  type={f.type === "integer" || f.type === "number" ? "number" : "text"} {...register(f.key)} />
              )}
              {f.description && <p className="mt-1 text-xs text-zinc-500">{f.description}</p>}
              {f.editable && !f.safeInPlace && <p className="mt-1 text-xs text-amber-700">Alterar este campo pode recriar o recurso.</p>}
              {err && <p className="mt-1 text-xs text-rose-700" role="alert">{err}</p>}
            </div>
          </div>
        );
      })}
      <div className="pt-2">
        <button type="submit" disabled={busy}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50">
          {submitLabel}
        </button>
      </div>
    </form>
  );
}
