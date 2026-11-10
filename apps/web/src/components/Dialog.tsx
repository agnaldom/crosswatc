"use client";

import { useEffect, useId, useRef } from "react";
import { Icon } from "@/components/Icon";

/** Native modal semantics provide focus containment, Escape and focus restoration. */
export function Dialog({ title, children, onClose, busy = false }: {
  title: string;
  children: React.ReactNode;
  onClose: () => void;
  busy?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  useEffect(() => {
    const dialog = ref.current;
    dialog?.showModal();
    return () => { dialog?.close(); };
  }, []);

  return (
    <dialog ref={ref} className="cw-dialog cw-workspace-page" aria-labelledby={titleId} aria-busy={busy}
      onCancel={(event) => { event.preventDefault(); if (!busy) onClose(); }}
      onClick={(event) => { if (event.target === event.currentTarget && !busy) onClose(); }}>
      <div className="cw-dialog-body">
        <div className="mb-5 flex items-center justify-between gap-4">
          <h2 id={titleId} className="text-base font-semibold tracking-tight">{title}</h2>
          <button type="button" className="cw-icon-button" onClick={onClose} disabled={busy} aria-label="Close dialog">
            <Icon name="cancel" className="h-4 w-4" aria-hidden />
          </button>
        </div>
        {children}
      </div>
    </dialog>
  );
}
