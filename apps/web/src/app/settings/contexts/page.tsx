"use client";

import { useState, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getContexts,
  setCurrentContext,
  checkContextConnection,
  addKubeConfig,
  removeContext,
} from "@/lib/api";

export default function ContextManagementPage() {
  const queryClient = useQueryClient();
  const contextsQuery = useQuery({ queryKey: ["contexts"], queryFn: getContexts });

  const [kubeConfigText, setKubeConfigText] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);
  const [contextToDelete, setContextToDelete] = useState<string | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<Record<string, { checking: boolean; connected?: boolean; error?: string }>>({});
  const fileInputRef = useRef<HTMLInputElement>(null);

  const switchMutation = useMutation({
    mutationFn: (context: string) => setCurrentContext(context),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["contexts"] });
      window.dispatchEvent(new CustomEvent("contextsUpdated"));
    },
  });

  const addMutation = useMutation({
    mutationFn: (yaml: string) => addKubeConfig(yaml),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["contexts"] });
      setKubeConfigText("");
      setValidationError(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
      window.dispatchEvent(new CustomEvent("contextsUpdated"));
    },
    onError: (err: Error) => {
      setValidationError(err.message);
    },
  });

  const removeMutation = useMutation({
    mutationFn: (context: string) => removeContext(context),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["contexts"] });
      setContextToDelete(null);
      window.dispatchEvent(new CustomEvent("contextsUpdated"));
    },
  });

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result;
      if (typeof content === "string") {
        setKubeConfigText(content);
        setValidationError(null);
      }
    };
    reader.readAsText(file);
  };

  const handleAddSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError(null);

    const text = kubeConfigText.trim();
    if (!text) {
      setValidationError("Kubeconfig content cannot be empty");
      return;
    }

    if (!text.includes("apiVersion:") || !text.includes("kind:")) {
      setValidationError("Invalid kubeconfig format: missing 'apiVersion:' or 'kind:'");
      return;
    }

    addMutation.mutate(text);
  };

  const handleTestConnection = async (contextName: string) => {
    setConnectionStatus((prev) => ({
      ...prev,
      [contextName]: { checking: true },
    }));

    try {
      const res = await checkContextConnection(contextName);
      setConnectionStatus((prev) => ({
        ...prev,
        [contextName]: { checking: false, connected: res.connected, error: res.error },
      }));
    } catch (err) {
      setConnectionStatus((prev) => ({
        ...prev,
        [contextName]: { checking: false, connected: false, error: (err as Error).message },
      }));
    }
  };

  const data = contextsQuery.data;
  const contexts = data?.contexts ?? ["in-cluster"];
  const currentContext = data?.currentContext ?? "in-cluster";
  const isInCluster = data?.isInCluster ?? false;

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-6">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            Context Management
          </h1>
          <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
            Inspect, switch, and upload Kubernetes cluster credentials and context configurations.
          </p>
        </div>
        <button
          type="button"
          onClick={() => contextsQuery.refetch()}
          disabled={contextsQuery.isFetching}
          className="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 bg-white px-3 py-1.5 text-xs font-semibold text-zinc-700 shadow-sm transition hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700"
        >
          <svg viewBox="0 0 20 20" fill="currentColor" className={`h-3.5 w-3.5 ${contextsQuery.isFetching ? "animate-spin" : ""}`}>
            <path fillRule="evenodd" d="M15.312 11.424a5.5 5.5 0 01-9.201 2.466l-.312-.311h2.433a.75.75 0 000-1.5H3.75a.75.75 0 00-.75.75v4.482a.75.75 0 001.5 0v-2.14l.32.32a7 7 0 0011.712-3.138.75.75 0 00-1.47-.329zm-10.624-2.848a5.5 5.5 0 019.201-2.466l.312.311H11.768a.75.75 0 000 1.5h4.482a.75.75 0 00.75-.75V2.689a.75.75 0 00-1.5 0v2.14l-.32-.32a7 7 0 00-11.712 3.138.75.75 0 001.47.329z" clipRule="evenodd" />
          </svg>
          <span>Refresh</span>
        </button>
      </div>

      {isInCluster && (
        <div className="rounded-lg border border-blue-200 bg-blue-50/70 p-4 dark:border-blue-900/60 dark:bg-blue-950/30">
          <div className="flex items-center gap-2 text-xs font-medium text-blue-800 dark:text-blue-300">
            <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clipRule="evenodd" />
            </svg>
            <span>Running in In-Cluster mode. Context is automatically provisioned via Pod ServiceAccount tokens.</span>
          </div>
        </div>
      )}

      {/* Existing Contexts */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            Configured Contexts ({contexts.length})
          </h2>
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          {contexts.map((ctxName) => {
            const isActive = ctxName === currentContext;
            const status = connectionStatus[ctxName];

            return (
              <div
                key={ctxName}
                className={`relative flex flex-col justify-between rounded-xl border p-4 shadow-sm transition ${
                  isActive
                    ? "border-blue-500 bg-blue-50/30 ring-1 ring-blue-500/20 dark:border-blue-500/80 dark:bg-blue-950/20"
                    : "border-zinc-200 bg-white hover:border-zinc-300 dark:border-zinc-800 dark:bg-[#0e1117] dark:hover:border-zinc-700"
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div
                      className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg font-bold text-sm ${
                        isActive
                          ? "bg-blue-600 text-white"
                          : "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300"
                      }`}
                    >
                      {ctxName.charAt(0).toUpperCase()}
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-xs font-bold text-zinc-900 dark:text-zinc-100">
                          {ctxName}
                        </span>
                        {isActive && (
                          <span className="rounded-full bg-blue-100 px-2 py-0.5 text-[10px] font-semibold text-blue-700 dark:bg-blue-900 dark:text-blue-200">
                            Active
                          </span>
                        )}
                      </div>
                      <div className="mt-0.5 flex items-center gap-1.5 text-[11px] text-zinc-500 dark:text-zinc-400">
                        <span
                          className={`h-2 w-2 rounded-full ${
                            status?.connected
                              ? "bg-emerald-500"
                              : status?.connected === false
                              ? "bg-rose-500"
                              : isActive
                              ? "bg-emerald-500"
                              : "bg-zinc-400"
                          }`}
                        />
                        <span>
                          {status?.checking
                            ? "Testing connection..."
                            : status?.connected
                            ? "Connected"
                            : status?.connected === false
                            ? "Unreachable"
                            : "Kubernetes context"}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                {status?.error && (
                  <p className="mt-2 truncate text-[10px] text-rose-600 dark:text-rose-400" title={status.error}>
                    {status.error}
                  </p>
                )}

                <div className="mt-4 flex items-center justify-between border-t border-zinc-100 pt-3 dark:border-zinc-800/80">
                  <button
                    type="button"
                    onClick={() => handleTestConnection(ctxName)}
                    disabled={status?.checking}
                    className="text-[11px] font-medium text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-200"
                  >
                    Test Connection
                  </button>

                  <div className="flex items-center gap-2">
                    {!isActive && !isInCluster && (
                      <button
                        type="button"
                        onClick={() => switchMutation.mutate(ctxName)}
                        disabled={switchMutation.isPending}
                        className="rounded-md bg-zinc-100 px-2.5 py-1 text-[11px] font-semibold text-zinc-800 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700"
                      >
                        Select
                      </button>
                    )}
                    {!isInCluster && contexts.length > 1 && (
                      <button
                        type="button"
                        onClick={() => setContextToDelete(ctxName)}
                        className="rounded-md p-1 text-zinc-400 hover:text-rose-600 dark:hover:text-rose-400"
                        title="Remove context"
                      >
                        <svg viewBox="0 0 20 20" fill="currentColor" className="h-3.5 w-3.5">
                          <path fillRule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C8.327 4.025 9.16 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clipRule="evenodd" />
                        </svg>
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Add Kubernetes Context */}
      {!isInCluster && (
        <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-[#0e1117]">
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            Add Kubernetes Context
          </h2>
          <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
            Upload a kubeconfig file or paste its YAML representation. Newly imported clusters and contexts will be merged safely into your local configuration.
          </p>

          <form onSubmit={handleAddSubmit} className="mt-4 space-y-4">
            <div className="flex items-center gap-3">
              <input
                ref={fileInputRef}
                type="file"
                accept=".yaml,.yml,.kubeconfig,.txt"
                onChange={handleFileSelect}
                className="hidden"
                id="kubeconfig-upload-input"
              />
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="inline-flex items-center gap-1.5 rounded-lg border border-zinc-300 bg-white px-3 py-1.5 text-xs font-semibold text-zinc-700 shadow-sm hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700"
              >
                <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4">
                  <path fillRule="evenodd" d="M4.5 2A1.5 1.5 0 003 3.5v13A1.5 1.5 0 004.5 18h11a1.5 1.5 0 001.5-1.5V7.621a1.5 1.5 0 00-.44-1.06l-4.12-4.122A1.5 1.5 0 0011.378 2H4.5zm2.25 8.5a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5zm0 3a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5z" clipRule="evenodd" />
                </svg>
                <span>Upload File</span>
              </button>
              <span className="text-xs text-zinc-400">or paste kubeconfig YAML directly below:</span>
            </div>

            <textarea
              rows={8}
              value={kubeConfigText}
              onChange={(e) => {
                setKubeConfigText(e.target.value);
                setValidationError(null);
              }}
              placeholder="apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: my-cluster
contexts:
- context:
    cluster: my-cluster
    user: my-user
  name: my-context
current-context: my-context
kind: Config
users:
- name: my-user
  user:
    token: ..."
              className="w-full rounded-lg border border-zinc-300 bg-white p-3 font-mono text-xs text-zinc-900 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
            />

            {validationError && (
              <div className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-300">
                {validationError}
              </div>
            )}

            <div className="flex justify-end">
              <button
                type="submit"
                disabled={addMutation.isPending || !kubeConfigText.trim()}
                className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-blue-500 disabled:opacity-50"
              >
                {addMutation.isPending ? "Adding Context..." : "Add Context"}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {contextToDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
          <div className="w-full max-w-sm rounded-xl border border-zinc-200 bg-white p-5 shadow-xl dark:border-zinc-800 dark:bg-[#12151c]">
            <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
              Remove Context
            </h2>
            <p className="mt-2 text-xs text-zinc-600 dark:text-zinc-400">
              Are you sure you want to remove context <strong className="text-zinc-900 dark:text-zinc-100">{contextToDelete}</strong>? Associated standalone credentials will also be cleaned up.
            </p>
            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setContextToDelete(null)}
                className="rounded-lg border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => removeMutation.mutate(contextToDelete)}
                disabled={removeMutation.isPending}
                className="rounded-lg bg-rose-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-rose-500 disabled:opacity-50"
              >
                {removeMutation.isPending ? "Removing..." : "Remove Context"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
