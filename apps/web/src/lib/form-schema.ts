import { z } from "zod";
import type { FormField } from "./types";

/** Builds a Zod validator mirroring the XRD constraints of the editable fields. */
export function buildSchema(fields: FormField[]) {
  const shape: Record<string, z.ZodType> = {};
  for (const f of fields) {
    if (!f.editable) continue;
    shape[f.key] = fieldSchema(f);
  }
  return z.object(shape);
}

function fieldSchema(f: FormField): z.ZodType {
  let s: z.ZodType;
  if (f.enum?.length) {
    const values = f.enum.map(String);
    s = z.string().refine((v) => values.includes(v), { message: `Value must be one of: ${values.join(", ")}` });
  } else if (f.type === "integer" || f.type === "number") {
    let n = z.coerce.number({ message: "Invalid number" });
    if (f.type === "integer") n = n.int("Must be an integer");
    if (f.minimum !== undefined) n = n.min(f.minimum, `Minimum ${f.minimum}`);
    if (f.maximum !== undefined) n = n.max(f.maximum, `Maximum ${f.maximum}`);
    s = n;
  } else if (f.type === "boolean") {
    s = z.boolean();
  } else {
    let str = z.string();
    if (f.minLength !== undefined) str = str.min(f.minLength, `Minimum ${f.minLength} characters`);
    if (f.maxLength !== undefined) str = str.max(f.maxLength, `Maximum ${f.maxLength} characters`);
    if (f.pattern) str = str.regex(new RegExp(f.pattern), `Invalid format (${f.pattern})`);
    if (f.required) str = str.min(1, "Required");
    s = str;
  }
  return f.required ? s : s.optional();
}

/** Keys whose submitted value differs from the current one. Numbers in enums are compared as strings. */
export function changedValues(fields: FormField[], current: Record<string, unknown>, next: Record<string, unknown>) {
  const out: Record<string, unknown> = {};
  for (const f of fields) {
    if (!f.editable || !(f.key in next)) continue;
    const a = current[f.key];
    const b = next[f.key];
    if (b === undefined || b === "") continue;
    if (String(a) !== String(b)) out[f.key] = b;
  }
  return out;
}
