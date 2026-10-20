import { describe, test } from "@e2e-dev/web";
import { expect } from "e2e";

describe("Provisioning Lifecycle Tracker Component", () => {
  test("renders 7-stage provisioning golden path in infrastructure master-detail", async ({ app, screen }) => {
    await app.open("/infrastructure");

    // Wait for resources table to load
    await expect(screen.getByRole("textbox", "Search resources")).toBeVisible();
    const table = screen.getByRole("table");
    await expect(table.getByText("db-payments-prod")).toBeVisible();

    // Select first resource row
    await table.getByText("db-payments-prod").tap();

    // Verify detail pane displays the Lifecycle tab
    const lifecycleTab = screen.getByRole("button", "Lifecycle");
    await expect(lifecycleTab).toBeVisible();

    // Click Lifecycle tab
    await lifecycleTab.tap();

    // Verify Provisioning Tracker elements
    await expect(screen.getByText("Rastreador de Provisionamento GitOps")).toBeVisible();
    await expect(screen.getByText("Linha do Tempo de Provisionamento (7 Etapas)")).toBeVisible();

    // Verify presence of all 7 stages
    await expect(screen.getByText("1. Backstage")).toBeVisible();
    await expect(screen.getByText("2. Git & PR")).toBeVisible();
    await expect(screen.getByText("3. CI & Políticas")).toBeVisible();
    await expect(screen.getByText("4. Argo CD")).toBeVisible();
    await expect(screen.getByText("5. Crossplane")).toBeVisible();
    await expect(screen.getByText("6. Nuvem Cloud")).toBeVisible();
    await expect(screen.getByText("7. Concluído")).toBeVisible();
  });

  test("allows inspecting steps, toggling failure simulation, and stepping through lifecycle", async ({ app, screen }) => {
    await app.open("/infrastructure");
    const table = screen.getByRole("table");
    await table.getByText("db-payments-prod").tap();

    // Switch to Lifecycle tab
    await screen.getByRole("button", "Lifecycle").tap();
    await expect(screen.getByText("Rastreador de Provisionamento GitOps")).toBeVisible();

    // Click on Step 4 (Argo CD)
    await screen.getByText("4. Argo CD").tap();
    await expect(screen.getByText("Sincronização GitOps no Cluster de Controle")).toBeVisible();
    await expect(screen.getByText("Responsável: Argo CD Central (ApplicationSet Reconciler)")).toBeVisible();

    // Click on Step 2 (Git & PR)
    await screen.getByText("2. Git & PR").tap();
    await expect(screen.getByText("Commit e Pull Request no Repositório GitOps")).toBeVisible();

    // Test simulation buttons
    const simulateBtn = screen.getByRole("button", "⚠️ Simular Alerta");
    await expect(simulateBtn).toBeVisible();
    await simulateBtn.tap();

    // Verify alert state triggered
    await expect(screen.getByText("Intervenção Recomendada")).toBeVisible();

    // Remove alert simulation
    await screen.getByRole("button", "⚠️ Remover Alerta").tap();
    await expect(screen.getByText("Ativo & Pronto")).toBeVisible();

    // Test complete all
    await screen.getByRole("button", "✓ Concluir Tudo").tap();
    await expect(screen.getByText("Etapa 7 de 7 (100%)")).toBeVisible();
  });

  test("navigates to dedicated lifecycle page via Overview button", async ({ app, browser, screen }) => {
    await app.open("/infrastructure");
    const table = screen.getByRole("table");
    await table.getByText("db-payments-prod").tap();

    // Switch to Overview tab
    await screen.getByRole("button", "Overview").tap();
    await expect(screen.getByText("GitOps & Rastreabilidade")).toBeVisible();

    // Test inline button to switch to Lifecycle tab
    const inlineBtn = screen.getByRole("button", "Ver Lifecycle no Painel ↓");
    await expect(inlineBtn).toBeVisible();
    await inlineBtn.tap();
    await expect(screen.getByText("Rastreador de Provisionamento GitOps")).toBeVisible();

    // Switch back to Overview and click full page link
    await screen.getByRole("button", "Overview").tap();
    const fullLink = screen.getByRole("link", "Abrir Lifecycle Completo ↗");
    await expect(fullLink).toBeVisible();
    await fullLink.tap();

    // Verify navigation to dedicated page with ?tab=Lifecycle
    await browser.waitForURL(/.*tab=Lifecycle.*/, { timeout: 15_000 });
    await expect(screen.getByText("Rastreador de Provisionamento GitOps")).toBeVisible();
    await expect(screen.getByText("Linha do Tempo de Provisionamento (7 Etapas)")).toBeVisible();
  });
});
