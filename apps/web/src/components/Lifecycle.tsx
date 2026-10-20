"use client";

import { useEffect, useState } from "react";
import type { Stage, StageView, TrackedResource } from "@/lib/types";

export interface LifecycleProps {
  stages?: StageView[];
  resource?: TrackedResource;
  onSelectStage?: (stage: Stage) => void;
  className?: string;
}

interface StepConfig {
  id: number;
  stage: Stage;
  shortTitle: string;
  fullTitle: string;
  actor: string;
  explanation: (r?: TrackedResource) => string;
  requesterAction: (r?: TrackedResource, failed?: boolean) => string;
  getMetadata: (r?: TrackedResource, stageView?: StageView) => { label: string; value: string; isLink?: boolean; href?: string }[];
  defaultLogs: (r?: TrackedResource) => string[];
}

const STEP_DEFINITIONS: StepConfig[] = [
  {
    id: 1,
    stage: "REQUEST",
    shortTitle: "1. Backstage",
    fullTitle: "Solicitação no Portal de Autoatendimento",
    actor: "Desenvolvedor via Backstage (Elostage Scaffolder)",
    explanation: (r) =>
      `O solicitante preenche o formulário corporativo homologado no Backstage para ${r?.kind || "o recurso de infraestrutura"}, sem precisar manipular YAML ou ter permissões manuais de cloud. O template gera o manifesto declarativo padronizado.`,
    requesterAction: () =>
      "Nenhuma ação pendente. Sua solicitação foi validada sintaticamente e enviada para versionamento no Git.",
    getMetadata: (r) => [
      { label: "Template Utilizado", value: `template-${(r?.kind || "infra").toLowerCase()}` },
      { label: "Squad Solicitante", value: r?.owner || "platform-engineering" },
      { label: "Namespace Kubernetes", value: r?.claimRef?.namespace || "sre" },
      { label: "Validação de Schema", value: "Aprovada (OpenAPI v3)" },
    ],
    defaultLogs: () => [
      "[INFO] Formulário submetido com sucesso no Backstage Scaffolder",
      "[INFO] Parâmetros validados contra o catálogo corporativo de infraestrutura",
      "[INFO] Disparando ação publish:github:pull-request no repositório de GitOps",
    ],
  },
  {
    id: 2,
    stage: "PULL_REQUEST",
    shortTitle: "2. Git & PR",
    fullTitle: "Commit e Pull Request no Repositório GitOps",
    actor: "GitHub Bot no repositório de GitOps",
    explanation: (r) =>
      `O Backstage cria automaticamente uma branch de infraestrutura no repositório ${r?.repository || "crosswatc-org/argocd"} e abre o Pull Request com a definição do Claim no path correspondente do ambiente ${r?.environment || "dev"}.`,
    requesterAction: (_r, failed) =>
      failed
        ? "O Pull Request foi fechado sem merge ou rejeitado. Revise os comentários no GitHub para reabrir."
        : "Acompanhe o link do Pull Request para verificar comentários ou aprovações solicitadas.",
    getMetadata: (r) => [
      { label: "Repositório GitOps", value: r?.repository || "crosswatc-org/argocd" },
      { label: "Caminho Git", value: r?.gitPath || `claims/${r?.environment || "dev"}/${r?.name || "recurso"}.yaml` },
      { label: "Branch Alvo", value: `${(r?.kind || "infra").toLowerCase()}/${r?.environment || "dev"}/${r?.name || "res"}` },
      {
        label: "Pull Request",
        value: r?.repository ? `https://github.com/${r.repository}/pull/112` : "PR #112",
        isLink: true,
        href: r?.repository ? `https://github.com/${r.repository}/pull/112` : undefined,
      },
    ],
    defaultLogs: (r) => [
      `[INFO] Branch criada com sucesso no repositório ${r?.repository || "crosswatc-org/argocd"}`,
      `[INFO] Manifesto claim.yaml commitado em ${r?.gitPath || "claims/dev/"}`,
      "[INFO] Pull Request aberto solicitando revisão dos CODEOWNERS",
    ],
  },
  {
    id: 3,
    stage: "CI_POLICIES",
    shortTitle: "3. CI & Políticas",
    fullTitle: "Validação Automatizada de CI & Políticas de Segurança",
    actor: "GitHub Actions + Open Policy Agent / Kyverno",
    explanation: () =>
      "A esteira de integração contínua roda checagens de sintaxe, lint, conformidade de tags corporativas, compliance com a LGPD/PCI e exige aprovação de Tech Leads.",
    requesterAction: (r, failed) =>
      failed
        ? "Validações de política falharam. Verifique os logs do GitHub Actions para ajustar tags ou parâmetros inválidos."
        : r?.environment === "prod"
        ? "Ambiente de Produção: certifique-se de que o Tech Lead e SRE aprovaram o Pull Request."
        : "Validações automatizadas aprovadas. Aguardando aprovação padrão de CODEOWNERS.",
    getMetadata: (r) => [
      { label: "Pipeline Status", value: "Success (Checks Passed)" },
      { label: "Policy Check", value: "OPA / Kyverno Compliant" },
      { label: "Tags Obrigatórias", value: "environment, owner, created_by (OK)" },
      { label: "Quórum CODEOWNERS", value: r?.environment === "prod" ? "2 Aprovações (OK)" : "1 Aprovação (OK)" },
    ],
    defaultLogs: () => [
      "[INFO] GitHub Actions runner iniciado (job: validate-infra-claim)",
      "[INFO] Verificando conformidade de tags e nomenclatura corporativa... OK",
      "[INFO] Policy-as-Code: nenhuma violação de segurança detectada",
      "[INFO] CODEOWNERS revisou e aprovou o Pull Request com sucesso",
      "[INFO] Pull Request mergeado na branch principal",
    ],
  },
  {
    id: 4,
    stage: "ARGO_CD",
    shortTitle: "4. Argo CD",
    fullTitle: "Sincronização GitOps no Cluster de Controle",
    actor: "Argo CD Central (ApplicationSet Reconciler)",
    explanation: (r) =>
      `O ApplicationSet do Argo CD detecta o novo commit no Git e sincroniza o Claim no cluster Kubernetes de destino, aplicando sync-options de proteção (Prune=false, IgnoreExtraneous) para o recurso ${r?.name || ""}.`,
    requesterAction: (_r, failed) =>
      failed
        ? "Falha ao sincronizar manifesto no cluster. Verifique a conectividade do cluster ou anotações no Argo CD."
        : "Aguardando sincronização do Argo CD. Você pode acompanhar pelo console central.",
    getMetadata: (r) => [
      { label: "Argo CD Application", value: `crossplane-${(r?.kind || "infra").toLowerCase()}-${r?.environment || "dev"}` },
      { label: "Sync Status", value: "Synced (Wave 4)" },
      { label: "Health Status", value: "Progressing -> Healthy" },
      { label: "Sync Policy", value: "Automated (SelfHeal: true, Prune Safe)" },
    ],
    defaultLogs: (r) => [
      "[INFO] Webhook recebido do GitHub: novo commit detectado",
      `[INFO] ApplicationSet disparou sincronização do aplicativo crossplane-${(r?.kind || "infra").toLowerCase()}`,
      `[INFO] Aplicando objeto Claim (${r?.kind || "Resource"}) no cluster Kubernetes`,
      "[INFO] Status do Argo CD: Synced com sucesso",
    ],
  },
  {
    id: 5,
    stage: "CROSSPLANE",
    shortTitle: "5. Crossplane",
    fullTitle: "Reconciliação & Expansão de Recursos no Crossplane",
    actor: "Crossplane Composite Resource (XR) Controller",
    explanation: (r) =>
      `O Crossplane captura o Claim, instancia a Composite Resource Definition (XRD) e aplica a Composition homologada (${r?.composition?.name || "default"}), gerando os Managed Resources reais da nuvem.`,
    requesterAction: (_r, failed) =>
      failed
        ? "Erro de reconciliação no Crossplane. Verifique as condições do Composite Resource no painel Status."
        : "Nenhuma ação necessária. A inteligência de plataforma está orquestrando as chamadas de API da nuvem.",
    getMetadata: (r) => [
      { label: "Composition", value: r?.composition?.name || `app-${(r?.kind || "infra").toLowerCase()}-${r?.provider || "cloud"}` },
      { label: "Crossplane Condition", value: r?.synced ? "SYNCED: True" : "SYNCED: InProgress" },
      { label: "Claim Ref", value: `${r?.claimRef?.namespace || "default"}/${r?.claimRef?.name || r?.name || "res"}` },
      { label: "Proteção ArgoCD", value: "Prune=false, IgnoreExtraneous (Injetados)" },
    ],
    defaultLogs: (r) => [
      `[INFO] Claim ${r?.name || ""} capturado pelo Crossplane Engine`,
      `[INFO] Composition selecionada com base no provider ${r?.provider || "cloud"}`,
      "[INFO] Injetando credenciais do ProviderConfig corporativo",
      "[INFO] Renderizando recursos gerenciados (Managed Resources) filhos no cluster",
    ],
  },
  {
    id: 6,
    stage: "CLOUD",
    shortTitle: "6. Nuvem Cloud",
    fullTitle: "Provisionamento Físico no Provedor Cloud",
    actor: "Upbound Cloud Provider Engine",
    explanation: (r) =>
      `O provedor (${r?.provider?.toUpperCase() || "CLOUD"}) emite requisições assíncronas na API da nuvem. O provedor aloca a infraestrutura física, redes, discos criptografados e firewalls na região ${r?.region || "padrão"}.`,
    requesterAction: (_r, failed) =>
      failed
        ? "Erro no provedor de nuvem (ex. limites de quota, rede ou autorização). Consulte o Runbook de Troubleshooting."
        : "Aguardando o provisionamento físico na nuvem. O tempo varia de acordo com a complexidade do recurso.",
    getMetadata: (r) => {
      const provider = r?.provider?.toLowerCase() || "azure";
      if (provider === "aws") {
        return [
          { label: "AWS Account / Role", value: "arn:aws:iam::123456789012:role/crossplane-role" },
          { label: "Região AWS", value: r?.region || "us-east-1" },
          { label: "VPC / Subnet", value: "vpc-prod-crossplane-01" },
          { label: "Status da API AWS", value: "Active / Available" },
        ];
      }
      if (provider === "gcp") {
        return [
          { label: "GCP Project", value: "crosswatc-cloud-dev-01" },
          { label: "Região / Zona GCP", value: r?.region || "us-central1-a" },
          { label: "VPC Network", value: "gcp-crossplane-vpc" },
          { label: "Resource Status", value: "RUNNING / READY" },
        ];
      }
      return [
        { label: "Azure Subscription", value: "Sub_Dev_TI (121129d5-3986-447b-8a52)" },
        { label: "Resource Group", value: r?.resourceGroup || `rg-${r?.environment || "dev"}-crossplane-01` },
        { label: "Região Azure", value: r?.region || "brazilsouth" },
        { label: "ARM ProvisioningState", value: "Succeeded" },
      ];
    },
    defaultLogs: (r) => [
      `[INFO] Provider ${r?.provider?.toUpperCase() || "CLOUD"} autenticado com sucesso via credencial gerenciada`,
      `[INFO] Emitindo chamadas de API para alocação na região ${r?.region || "padrão"}...`,
      `[INFO] Nuvem provisionou a infraestrutura física com sucesso (Status 200 OK)`,
      "[INFO] Endpoints e metadados de conectividade recuperados da nuvem",
    ],
  },
  {
    id: 7,
    stage: "OPERATIONAL",
    shortTitle: "7. Concluído",
    fullTitle: "Pronto para Consumo & Injeção de Segredos",
    actor: "External Secrets Operator + Backstage Catalog",
    explanation: (r) =>
      `O recurso ${r?.name || ""} está 100% ativo e pronto para uso! As credenciais de acesso foram salvas de forma segura no Secret do namespace da squad e o recurso foi registrado no catálogo do Backstage.`,
    requesterAction: () =>
      "Pronto! Sua aplicação já pode consumir este recurso. Conecte seu workload ao Secret de conexão correspondente.",
    getMetadata: (r) => [
      { label: "Status Final", value: "✓ ATIVO E OPERACIONAL" },
      { label: "Secret Gerado", value: `${r?.name || "resource"}-connection (namespace: ${r?.claimRef?.namespace || "sre"})` },
      { label: "Backstage Catalog", value: "Sincronizado (Disponível na busca corporativa)" },
      { label: "Monitoramento", value: "Observabilidade & Datadog Ativos" },
    ],
    defaultLogs: (r) => [
      `[INFO] Secret de conexão injetado no namespace ${r?.claimRef?.namespace || "sre"}`,
      "[INFO] Catálogo do Backstage sincronizado com a entidade live",
      "[INFO] Monitoramento corporativo e tags de compliance validadas",
      `[SUCCESS] Provisionamento de ${r?.name || "recurso"} concluído com sucesso em 100% conformidade!`,
    ],
  },
];

export function Lifecycle({ stages, resource, onSelectStage, className = "" }: LifecycleProps) {
  // Determine current active step (1 to 7) based on stages or resource state
  const computedActiveStep = (() => {
    if (stages && stages.length > 0) {
      // Find the first stage that is in progress or failed
      const activeIdx = stages.findIndex((s) => s.status === "IN_PROGRESS" || s.status === "FAILED");
      if (activeIdx >= 0) return activeIdx + 1;
      // If all succeeded
      const allSucceeded = stages.every((s) => s.status === "SUCCEEDED" || s.status === "SKIPPED");
      if (allSucceeded) return 7;
    }
    if (resource?.resourceState === "OPERATIONAL" || resource?.ready) {
      return 7;
    }
    return 5; // Default to Crossplane stage for provisioning resources
  })();

  const [stepOverride, setStepOverride] = useState<number | null>(null);
  const [selectedOverride, setSelectedOverride] = useState<number | null>(null);
  const currentStep = stepOverride ?? computedActiveStep;
  const selectedStepId = selectedOverride ?? currentStep;
  const [isSimulatedFailure, setIsSimulatedFailure] = useState(false);
  const [clock, setClock] = useState("Atualizado agora");
  const [logTimestamps, setLogTimestamps] = useState<string[]>([]);

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      setClock(`Atualizado às ${now.toLocaleTimeString()}`);
      const t = now.getTime();
      setLogTimestamps([
        new Date(t - 140000).toLocaleTimeString(),
        new Date(t - 105000).toLocaleTimeString(),
        new Date(t - 70000).toLocaleTimeString(),
        new Date(t - 35000).toLocaleTimeString(),
        now.toLocaleTimeString(),
      ]);
    };
    updateTime();
    const timer = setInterval(updateTime, 3000);
    return () => clearInterval(timer);
  }, []);

  const handleStepSelect = (stepId: number) => {
    setSelectedOverride(stepId);
    const stepConfig = STEP_DEFINITIONS.find((s) => s.id === stepId);
    if (stepConfig && onSelectStage) {
      onSelectStage(stepConfig.stage);
    }
  };

  const handlePrevStep = () => {
    if (currentStep > 1) {
      const prev = currentStep - 1;
      setStepOverride(prev);
      setSelectedOverride(prev);
      setIsSimulatedFailure(false);
    }
  };

  const handleNextStep = () => {
    if (currentStep < 7) {
      const next = currentStep + 1;
      setStepOverride(next);
      setSelectedOverride(next);
      setIsSimulatedFailure(false);
    }
  };

  const handleToggleFailure = () => {
    setIsSimulatedFailure((prev) => !prev);
  };

  const handleCompleteAll = () => {
    setStepOverride(7);
    setSelectedOverride(7);
    setIsSimulatedFailure(false);
  };

  const selectedStep = STEP_DEFINITIONS.find((s) => s.id === selectedStepId) || STEP_DEFINITIONS[0];
  const selectedStageView = stages?.find((s) => s.stage === selectedStep.stage);
  const percentage = Math.round((currentStep / 7) * 100);

  // External URLs
  const prUrl = resource?.repository
    ? `https://github.com/${resource.repository}/pull/112`
    : "https://github.com/crosswatc-org/argocd/pull/112";
  const argoUrl = "https://argocd.elo.corp";
  const backstageUrl = resource
    ? `http://localhost:3000/catalog/default/resource/${resource.name}`
    : "http://localhost:3000/catalog";
  const cloudUrl =
    resource?.provider?.toLowerCase() === "aws"
      ? "https://console.aws.amazon.com"
      : resource?.provider?.toLowerCase() === "gcp"
      ? "https://console.cloud.google.com"
      : "https://portal.azure.com";

  return (
    <div className={`space-y-6 ${className}`} data-testid="provisioning-tracker">
      {/* ========================================================================= */}
      {/* PLATFORM HEADER & CONTEXT BAR */}
      {/* ========================================================================= */}
      <div className="flex flex-col gap-4 rounded-2xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#161b22] md:flex-row md:items-center md:justify-between">
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-full border border-blue-500/30 bg-blue-500/10 px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400">
              Golden Path
            </span>
            <span className="rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
              Visão do Solicitante
            </span>
            <span className="rounded-full border border-cyan-500/30 bg-cyan-500/10 px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wider text-cyan-600 dark:text-cyan-400">
              Crossplane Control Plane
            </span>
            {resource?.provider && (
              <span className="rounded-full border border-purple-500/30 bg-purple-500/10 px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wider text-purple-600 dark:text-purple-400">
                {resource.provider}
              </span>
            )}
          </div>
          <h2 className="text-xl font-bold tracking-tight text-zinc-900 dark:text-white sm:text-2xl">
            Rastreador de Provisionamento GitOps
          </h2>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            Acompanhamento ponta a ponta do ciclo de vida da infraestrutura solicitada — do Backstage à Nuvem
          </p>
        </div>

        {/* Live Clock & Reconciliation Status */}
        <div className="flex items-center gap-3 self-start rounded-xl border border-zinc-200 bg-zinc-50/80 px-4 py-2.5 dark:border-zinc-800 dark:bg-zinc-900/60 md:self-auto">
          <span className="relative flex h-2.5 w-2.5">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
            <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-500" />
          </span>
          <div className="text-xs">
            <div className="font-semibold text-zinc-800 dark:text-zinc-200">Reconciliação Contínua</div>
            <div className="text-[11px] text-zinc-500 dark:text-zinc-400">{clock}</div>
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* SUMMARY CARD & SIMULATION CONTROLS */}
      {/* ========================================================================= */}
      <div className="space-y-4 rounded-2xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#161b22]">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-0.5">
            <span className="text-[11px] font-bold uppercase tracking-wider text-zinc-400">Recurso Monitorado</span>
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-bold text-zinc-900 dark:text-white">
                {resource?.name || "recurso-selecionado"}
              </span>
              <span className="text-xs text-zinc-500">({resource?.kind || "CompositeResource"})</span>
            </div>
          </div>

          {/* Interactive Stepper Navigation / Simulation Controls */}
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={handlePrevStep}
              disabled={currentStep <= 1}
              className="rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-1.5 text-xs font-semibold text-zinc-700 transition hover:bg-zinc-100 disabled:opacity-40 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700"
            >
              ◀ Etapa Anterior
            </button>
            <button
              type="button"
              onClick={handleNextStep}
              disabled={currentStep >= 7}
              className="rounded-lg bg-cyan-600 px-3 py-1.5 text-xs font-semibold text-white shadow transition hover:bg-cyan-500 disabled:opacity-40"
            >
              Próxima Etapa ▶
            </button>
            <button
              type="button"
              onClick={handleToggleFailure}
              className={`rounded-lg border px-3 py-1.5 text-xs font-semibold transition ${
                isSimulatedFailure
                  ? "border-rose-500 bg-rose-500 text-white"
                  : "border-rose-300 bg-rose-50 text-rose-700 hover:bg-rose-100 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-300 dark:hover:bg-rose-900/60"
              }`}
            >
              ⚠️ {isSimulatedFailure ? "Remover Alerta" : "Simular Alerta"}
            </button>
            <button
              type="button"
              onClick={handleCompleteAll}
              className="rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700 transition hover:bg-emerald-100 dark:border-emerald-800/80 dark:bg-emerald-950/40 dark:text-emerald-300 dark:hover:bg-emerald-900/60"
            >
              ✓ Concluir Tudo
            </button>
          </div>
        </div>

        {/* Resumo Detalhado dos Metadados */}
        <div className="grid grid-cols-2 gap-3 rounded-xl border border-zinc-100 bg-zinc-50/80 p-3.5 text-xs dark:border-zinc-800/80 dark:bg-zinc-900/40 sm:grid-cols-3 lg:grid-cols-6">
          <div>
            <span className="block text-[11px] text-zinc-400">ID / Nome</span>
            <span className="font-semibold text-zinc-900 dark:text-zinc-100">{resource?.name || "—"}</span>
          </div>
          <div>
            <span className="block text-[11px] text-zinc-400">Tipo de Recurso</span>
            <span className="font-medium text-zinc-800 dark:text-zinc-200">{resource?.kind || "—"}</span>
          </div>
          <div>
            <span className="block text-[11px] text-zinc-400">Ambiente</span>
            <span className="inline-block rounded bg-cyan-100 px-2 py-0.5 font-mono text-[11px] font-semibold text-cyan-800 dark:bg-cyan-950/60 dark:text-cyan-300">
              {resource?.environment || "dev"}
            </span>
          </div>
          <div>
            <span className="block text-[11px] text-zinc-400">Solicitante / Squad</span>
            <span className="font-medium text-zinc-800 dark:text-zinc-200">{resource?.owner || "platform"}</span>
          </div>
          <div>
            <span className="block text-[11px] text-zinc-400">Região / RG</span>
            <span className="block truncate font-mono text-[11px] text-zinc-700 dark:text-zinc-300" title={resource?.region || resource?.resourceGroup}>
              {resource?.region || resource?.resourceGroup || "brazilsouth"}
            </span>
          </div>
          <div>
            <span className="block text-[11px] text-zinc-400">Status Geral</span>
            {isSimulatedFailure ? (
              <span className="inline-flex items-center gap-1.5 rounded-full border border-rose-500/30 bg-rose-500/10 px-2 py-0.5 font-bold text-rose-600 dark:text-rose-400">
                <span className="h-1.5 w-1.5 rounded-full bg-rose-500" />
                Intervenção Recomendada
              </span>
            ) : currentStep === 7 ? (
              <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 font-bold text-emerald-600 dark:text-emerald-400">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                Ativo & Pronto
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 rounded-full border border-cyan-500/30 bg-cyan-500/10 px-2 py-0.5 font-bold text-cyan-600 dark:text-cyan-400">
                <span className="h-1.5 w-1.5 rounded-full bg-cyan-500 animate-pulse" />
                Em Provisionamento
              </span>
            )}
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* 7-STAGE PROGRESS STEPPER */}
      {/* ========================================================================= */}
      <div className="rounded-2xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#161b22]">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-sm font-bold text-zinc-900 dark:text-white">
            Linha do Tempo de Provisionamento (7 Etapas)
          </h3>
          <span className="rounded-full border border-cyan-500/30 bg-cyan-500/10 px-2.5 py-0.5 text-xs font-bold text-cyan-700 dark:text-cyan-300">
            Etapa {currentStep} de 7 ({percentage}%)
          </span>
        </div>

        {/* Continuous Progress Bar */}
        <div className="mb-6 h-2 w-full overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
          <div
            className="h-full bg-gradient-to-r from-cyan-500 via-blue-500 to-emerald-500 transition-all duration-500"
            style={{ width: `${percentage}%` }}
          />
        </div>

        {/* 7 Interactive Step Cards */}
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 md:grid-cols-4 lg:grid-cols-7" role="tablist" aria-label="Etapas de Provisionamento">
          {STEP_DEFINITIONS.map((step) => {
            const isCompleted = step.id < currentStep;
            const isCurrent = step.id === currentStep;
            const isSelected = step.id === selectedStepId;
            const hasFailure = isSimulatedFailure && isCurrent;

            let cardStyles = "border-zinc-200 bg-zinc-50/70 text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/40 dark:text-zinc-400";
            let badgeBg = "bg-zinc-200 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400";
            let badgeText = "Pendente";
            let badgeIcon = "○";

            if (hasFailure) {
              cardStyles = "border-rose-400 bg-rose-50/80 text-rose-800 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-300 shadow-sm";
              badgeBg = "bg-rose-100 text-rose-700 dark:bg-rose-900/60 dark:text-rose-300";
              badgeText = "Atenção";
              badgeIcon = "⚠️";
            } else if (isCompleted) {
              cardStyles = "border-emerald-300 bg-emerald-50/70 text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-950/20 dark:text-emerald-300";
              badgeBg = "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/60 dark:text-emerald-300";
              badgeText = "Concluído";
              badgeIcon = "✓";
            } else if (isCurrent) {
              cardStyles = "border-cyan-400 bg-cyan-50/80 text-cyan-800 dark:border-cyan-700 dark:bg-cyan-950/40 dark:text-cyan-200 shadow-sm";
              badgeBg = "bg-cyan-100 text-cyan-700 dark:bg-cyan-900/60 dark:text-cyan-300";
              badgeText = "Em Andamento";
              badgeIcon = "●";
            }

            const activeRing = isSelected ? "ring-2 ring-cyan-500 ring-offset-2 ring-offset-white dark:ring-offset-[#161b22]" : "";

            return (
              <button
                key={step.id}
                type="button"
                role="tab"
                aria-selected={isSelected}
                onClick={() => handleStepSelect(step.id)}
                className={`flex flex-col justify-between rounded-xl border p-3 text-left transition hover:-translate-y-0.5 ${cardStyles} ${activeRing}`}
              >
                <div>
                  <div className="mb-2 flex items-center justify-between">
                    <span
                      className={`flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold ${
                        step.id <= currentStep ? "bg-cyan-600 text-white" : "bg-zinc-300 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-400"
                      }`}
                    >
                      {step.id}
                    </span>
                    <span className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${badgeBg}`}>
                      {badgeIcon} {badgeText}
                    </span>
                  </div>
                  <div className="truncate text-xs font-bold text-zinc-900 dark:text-zinc-100" title={step.fullTitle}>
                    {step.shortTitle}
                  </div>
                </div>
                <div className="mt-3 flex items-center justify-between text-[11px] font-medium opacity-80">
                  <span>Ver detalhes</span>
                  <span>→</span>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* ========================================================================= */}
      {/* DETALHES DA ETAPA SELECIONADA + QUICK LINKS / RUNBOOK */}
      {/* ========================================================================= */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Coluna da Esquerda (2 spans): O que está acontecendo + Ações + Evidências + Logs */}
        <div className="space-y-6 lg:col-span-2">
          {/* Card de Detalhamento da Etapa */}
          <div className="space-y-4 rounded-2xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-[#161b22]">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-100 pb-4 dark:border-zinc-800">
              <div className="flex items-center gap-3">
                <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-cyan-600 text-sm font-bold text-white shadow">
                  {selectedStep.id}
                </span>
                <div>
                  <h4 className="text-base font-bold text-zinc-900 dark:text-white sm:text-lg">
                    {selectedStep.fullTitle}
                  </h4>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400">
                    Responsável: {selectedStep.actor}
                  </p>
                </div>
              </div>

              {/* Status Badge da Etapa */}
              {isSimulatedFailure && selectedStep.id === currentStep ? (
                <span className="flex items-center gap-1.5 rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-bold text-rose-600 dark:text-rose-400">
                  <span className="h-2 w-2 rounded-full bg-rose-500 animate-pulse" />
                  Bloqueio / Atenção
                </span>
              ) : selectedStep.id < currentStep ? (
                <span className="flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-bold text-emerald-600 dark:text-emerald-400">
                  <span className="h-2 w-2 rounded-full bg-emerald-500" />
                  Concluído com Sucesso
                </span>
              ) : selectedStep.id === currentStep ? (
                <span className="flex items-center gap-1.5 rounded-full border border-cyan-500/30 bg-cyan-500/10 px-3 py-1 text-xs font-bold text-cyan-600 dark:text-cyan-400">
                  <span className="h-2 w-2 rounded-full bg-cyan-500 animate-pulse" />
                  Em Andamento na Nuvem
                </span>
              ) : (
                <span className="flex items-center gap-1.5 rounded-full border border-zinc-300 bg-zinc-100 px-3 py-1 text-xs font-bold text-zinc-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400">
                  ○ Aguardando Etapas Anteriores
                </span>
              )}
            </div>

            {/* O que está acontecendo agora */}
            <div className="rounded-xl border border-zinc-100 bg-zinc-50/70 p-4 space-y-1.5 dark:border-zinc-800/80 dark:bg-zinc-900/40">
              <div className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider text-zinc-400">
                <svg className="h-4 w-4 text-cyan-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                O que está acontecendo agora:
              </div>
              <p className="text-sm leading-relaxed text-zinc-800 dark:text-zinc-200">
                {selectedStageView?.message || selectedStep.explanation(resource)}
              </p>
              {selectedStageView?.technicalDetail && (
                <div className="mt-2 rounded bg-zinc-100 p-2 font-mono text-[11px] text-zinc-700 dark:bg-zinc-950 dark:text-zinc-300">
                  {selectedStageView.technicalDetail}
                </div>
              )}
            </div>

            {/* Próxima Ação Necessária do Solicitante */}
            <div className="flex items-start gap-3 rounded-xl border border-cyan-200 bg-cyan-50/50 p-4 dark:border-cyan-900/60 dark:bg-cyan-950/20">
              <div className="mt-0.5 shrink-0 rounded-lg bg-cyan-600/10 p-2 text-cyan-600 dark:text-cyan-400">
                <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div>
                <div className="text-xs font-bold uppercase tracking-wider text-cyan-800 dark:text-cyan-300">
                  Ação requerida de você (Solicitante):
                </div>
                <p className="mt-0.5 text-sm text-zinc-700 dark:text-zinc-300">
                  {selectedStep.requesterAction(resource, isSimulatedFailure && selectedStep.id === currentStep)}
                </p>
              </div>
            </div>

            {/* Metadados e Evidências Técnicas */}
            <div className="border-t border-zinc-100 pt-4 dark:border-zinc-800">
              <div className="mb-3 text-xs font-bold uppercase tracking-wider text-zinc-400">
                Evidências e Metadados Técnicos:
              </div>
              <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 text-xs">
                {selectedStep.getMetadata(resource, selectedStageView).map((m, idx) => (
                  <div key={idx} className="rounded-lg border border-zinc-100 bg-zinc-50/60 p-2.5 dark:border-zinc-800 dark:bg-zinc-900/30">
                    <span className="block text-[11px] text-zinc-400">{m.label}</span>
                    {m.isLink && m.href ? (
                      <a
                        href={m.href}
                        target="_blank"
                        rel="noreferrer"
                        className="font-medium text-cyan-600 hover:underline dark:text-cyan-400 break-all"
                      >
                        {m.value} ↗
                      </a>
                    ) : (
                      <span className="font-medium text-zinc-800 dark:text-zinc-200 break-all">{m.value}</span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Terminal de Logs / Eventos em Tempo Real */}
          <div className="overflow-hidden rounded-2xl border border-zinc-200 bg-zinc-950 text-zinc-300 shadow-sm dark:border-zinc-800">
            <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-900/80 px-4 py-3">
              <div className="flex items-center gap-2">
                <span className="h-3 w-3 rounded-full bg-rose-500/80" />
                <span className="h-3 w-3 rounded-full bg-amber-500/80" />
                <span className="h-3 w-3 rounded-full bg-emerald-500/80" />
                <span className="ml-2 font-mono text-xs text-zinc-400">Audit Logs & Tracing em Tempo Real</span>
              </div>
              <span className="font-mono text-[11px] text-zinc-500">
                {selectedStep.defaultLogs(resource).length + (isSimulatedFailure && selectedStep.id === currentStep ? 1 : 0)} eventos
              </span>
            </div>
            <div className="max-h-56 space-y-1.5 overflow-y-auto p-4 font-mono text-xs text-zinc-300">
              {selectedStep.defaultLogs(resource).map((log, idx) => {
                let color = "text-zinc-300";
                if (log.includes("[INFO]")) color = "text-cyan-300";
                if (log.includes("[SUCCESS]")) color = "text-emerald-400 font-bold";
                if (log.includes("[WARN]")) color = "text-amber-400";
                if (log.includes("[ERROR]")) color = "text-rose-400 font-bold";

                const ts = logTimestamps[idx] ?? "12:00:00";

                return (
                  <div key={idx} className={color}>
                    <span className="text-zinc-500">[{ts}]</span> {log}
                  </div>
                );
              })}

              {isSimulatedFailure && selectedStep.id === currentStep && (
                <div className="rounded border border-rose-800/80 bg-rose-950/40 p-2 font-bold text-rose-300">
                  <span className="text-zinc-500">[{logTimestamps[4] ?? "12:00:00"}]</span> [WARN] NicInUse / ReconcilePending:
                  Interface associada a recurso em conflito. Verifique anotações de proteção no ApplicationSet.
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Coluna da Direita (1 span): Links Úteis, SLA & Runbook */}
        <div className="space-y-6">
          {/* Links Rápidos e Acessos Diretos */}
          <div className="space-y-4 rounded-2xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#161b22]">
            <h4 className="text-xs font-bold uppercase tracking-wider text-zinc-400">
              Links e Acessos Diretos
            </h4>
            <div className="space-y-2">
              <a
                href={prUrl}
                target="_blank"
                rel="noreferrer"
                className="group flex items-center justify-between rounded-xl border border-zinc-100 bg-zinc-50/70 p-3 text-xs transition hover:border-zinc-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/40 dark:hover:border-zinc-700 dark:hover:bg-zinc-800"
              >
                <div className="flex items-center gap-2.5">
                  <svg className="h-4 w-4 text-zinc-500 group-hover:text-zinc-900 dark:text-zinc-400 dark:group-hover:text-white" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                  </svg>
                  <span className="font-medium text-zinc-800 dark:text-zinc-200">Manifestos no GitHub</span>
                </div>
                <span className="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200">↗</span>
              </a>

              <a
                href={argoUrl}
                target="_blank"
                rel="noreferrer"
                className="group flex items-center justify-between rounded-xl border border-zinc-100 bg-zinc-50/70 p-3 text-xs transition hover:border-zinc-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/40 dark:hover:border-zinc-700 dark:hover:bg-zinc-800"
              >
                <div className="flex items-center gap-2.5">
                  <svg className="h-4 w-4 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                  </svg>
                  <span className="font-medium text-zinc-800 dark:text-zinc-200">Console do Argo CD</span>
                </div>
                <span className="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200">↗</span>
              </a>

              <a
                href={backstageUrl}
                target="_blank"
                rel="noreferrer"
                className="group flex items-center justify-between rounded-xl border border-zinc-100 bg-zinc-50/70 p-3 text-xs transition hover:border-zinc-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/40 dark:hover:border-zinc-700 dark:hover:bg-zinc-800"
              >
                <div className="flex items-center gap-2.5">
                  <svg className="h-4 w-4 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                  </svg>
                  <span className="font-medium text-zinc-800 dark:text-zinc-200">Catálogo Backstage</span>
                </div>
                <span className="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200">↗</span>
              </a>

              <a
                href={cloudUrl}
                target="_blank"
                rel="noreferrer"
                className="group flex items-center justify-between rounded-xl border border-zinc-100 bg-zinc-50/70 p-3 text-xs transition hover:border-zinc-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/40 dark:hover:border-zinc-700 dark:hover:bg-zinc-800"
              >
                <div className="flex items-center gap-2.5">
                  <svg className="h-4 w-4 text-blue-500" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z" />
                  </svg>
                  <span className="font-medium text-zinc-800 dark:text-zinc-200">
                    Console da Nuvem ({resource?.provider?.toUpperCase() || "CLOUD"})
                  </span>
                </div>
                <span className="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200">↗</span>
              </a>
            </div>
          </div>

          {/* SLA e Estimativa de Tempo */}
          <div className="space-y-3 rounded-2xl border border-zinc-200 bg-white p-5 shadow-sm dark:border-zinc-800 dark:bg-[#161b22]">
            <h4 className="text-xs font-bold uppercase tracking-wider text-zinc-400">
              Estimativa de Tempo (SLA)
            </h4>
            <div className="space-y-2 text-xs">
              <div className="flex justify-between text-zinc-500 dark:text-zinc-400">
                <span>Tempo Decorrido:</span>
                <span className="font-mono font-bold text-zinc-900 dark:text-zinc-100">
                  {currentStep === 7 ? "4m 12s" : "2m 45s"}
                </span>
              </div>
              <div className="flex justify-between text-zinc-500 dark:text-zinc-400">
                <span>Tempo Médio Total:</span>
                <span className="font-mono text-zinc-800 dark:text-zinc-200">~ 5m 30s</span>
              </div>
              <div className="flex justify-between text-zinc-500 dark:text-zinc-400">
                <span>Nível de Automação:</span>
                <span className="font-semibold text-emerald-600 dark:text-emerald-400">100% Declarativo (Zero CLI)</span>
              </div>
            </div>

            <div className="rounded-xl border border-zinc-100 bg-zinc-50/70 p-3 text-[11px] leading-relaxed text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900/40 dark:text-zinc-400">
              ℹ️ Recursos de banco de dados e clusters gerenciados (ex. <strong>PostgreSQL</strong> e <strong>AKS</strong>) levam de 4 a 12 minutos na nuvem. Storage Accounts e VMs levam em média de 2 a 5 minutos.
            </div>
          </div>

          {/* Runbook & Troubleshooting de Auto-recuperação */}
          <div className="space-y-2 rounded-2xl border border-indigo-200 bg-gradient-to-br from-indigo-50/60 to-white p-5 shadow-sm dark:border-indigo-900/50 dark:bg-gradient-to-br dark:from-indigo-950/30 dark:to-[#161b22]">
            <div className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider text-indigo-700 dark:text-indigo-300">
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
              </svg>
              Dúvidas ou Erros?
            </div>
            <p className="text-xs leading-relaxed text-zinc-700 dark:text-zinc-300">
              Consulte o <strong>Runbook de Troubleshooting da Plataforma</strong> com os casos catalogados de reconciliação e auto-recuperação (self-healing).
            </p>
            <div className="pt-1">
              <span className="inline-block rounded-md bg-indigo-100 px-2.5 py-1 text-xs font-semibold text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-300">
                TROUBLESHOOTING_RUNBOOK.md
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
