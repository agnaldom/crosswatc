package lifecycle

import (
	"fmt"

	"github.com/agnaldom/crosswatc/internal/domain"
)

// ResolveStages computes the 7-stage journey (§31). Stage N is only evaluated if
// N-1 is SUCCEEDED or SKIPPED; after the first FAILED (or not-done) stage the rest are PENDING.
func ResolveStages(s Signals) []domain.StageView {
	resolvers := []func(Signals) domain.StageView{
		requestStage, pullRequestStage, ciStage, argoStage, crossplaneStage, cloudStage, operationalStage,
	}
	out := make([]domain.StageView, 0, len(resolvers))
	blocked := false
	for i, r := range resolvers {
		if blocked {
			out = append(out, domain.StageView{Stage: domain.Stages[i], Status: domain.StagePending})
			continue
		}
		v := r(s)
		out = append(out, v)
		if !v.Status.Done() {
			blocked = true
		}
	}
	return out
}

func requestStage(s Signals) domain.StageView {
	v := domain.StageView{Stage: domain.StageRequest}
	switch {
	case s.Backstage.TaskID == "":
		v.Status, v.Message = domain.StageSkipped, "Solicitação criada fora do Portal"
	case s.Backstage.Status == "completed":
		v.Status, v.Message = domain.StageSucceeded, "Solicitação registrada no Portal"
	case s.Backstage.Status == "failed" || s.Backstage.Status == "cancelled":
		v.Status, v.ActionOwner = domain.StageFailed, domain.OwnerRequester
		v.Message = "A solicitação no Portal não foi concluída"
		v.TechnicalDetail = "scaffolder task " + s.Backstage.TaskID + " " + s.Backstage.Status
	default:
		v.Status, v.ActionOwner, v.Message = domain.StageInProgress, domain.OwnerPlatform, "Processando solicitação no Portal"
	}
	return v
}

// clusterOnly reports a resource observed in the cluster without PR information.
func clusterOnly(s Signals) bool { return !s.PR.Found && (s.Claim.Present || s.Argo.Found) }

func pullRequestStage(s Signals) domain.StageView {
	v := domain.StageView{Stage: domain.StagePullRequest}
	switch {
	case s.PR.Found && !s.PR.Open && !s.PR.Merged:
		v.Status, v.ActionOwner = domain.StageFailed, domain.OwnerRequester
		v.Message = fmt.Sprintf("Pull Request #%d foi fechado sem merge", s.PR.Number)
	case s.PR.Found:
		v.Status, v.Message = domain.StageSucceeded, fmt.Sprintf("Pull Request #%d criado", s.PR.Number)
	case clusterOnly(s):
		v.Status, v.Message = domain.StageSkipped, "Pull Request não localizado; estado obtido do cluster"
	default:
		v.Status, v.ActionOwner, v.Message = domain.StageInProgress, domain.OwnerPlatform, "Aguardando criação do Pull Request"
	}
	return v
}

func ciStage(s Signals) domain.StageView {
	v := domain.StageView{Stage: domain.StageCIPolicies}
	switch {
	case clusterOnly(s):
		v.Status = domain.StageSkipped
	case s.PR.Merged:
		v.Status, v.Message = domain.StageSucceeded, "Validações aprovadas e mudança aprovada"
	case s.PR.Checks == ChecksFailure:
		v.Status, v.ActionOwner = domain.StageFailed, domain.OwnerRequester
		v.Message = "Validações automáticas falharam; corrija a configuração"
	case s.PR.ChangesRequested:
		v.Status, v.ActionOwner = domain.StageInProgress, domain.OwnerRequester
		v.Message = "Revisor pediu alterações"
	case s.PR.Checks == ChecksSuccess && !s.PR.Approved:
		v.Status, v.ActionOwner, v.Message = domain.StageInProgress, domain.OwnerApprover, "Aguardando aprovação"
	case s.PR.Checks == ChecksSuccess && s.PR.Approved:
		v.Status, v.ActionOwner, v.Message = domain.StageInProgress, domain.OwnerApprover, "Aprovado; aguardando merge"
	default:
		v.Status, v.ActionOwner, v.Message = domain.StageInProgress, domain.OwnerPlatform, "Executando validações"
	}
	return v
}

func argoStage(s Signals) domain.StageView {
	v := domain.StageView{Stage: domain.StageArgoCD}
	switch {
	case !s.Argo.Found && s.Claim.Present:
		v.Status, v.Message = domain.StageSkipped, "Aplicação Argo CD não identificada"
	case s.argoFailed():
		v.Status, v.ActionOwner = domain.StageFailed, domain.OwnerPlatform
		v.Message, v.TechnicalDetail = "Falha ao sincronizar com o cluster", s.Argo.Message
	case s.argoOnCommit():
		v.Status, v.Message = domain.StageSucceeded, "Configuração sincronizada"
	default:
		v.Status, v.ActionOwner, v.Message = domain.StageInProgress, domain.OwnerPlatform, "Sincronizando configuração"
		v.TechnicalDetail = fmt.Sprintf("sync=%s phase=%s", s.Argo.SyncStatus, s.Argo.OperationPhase)
	}
	return v
}

func crossplaneStage(s Signals) domain.StageView {
	v := domain.StageView{Stage: domain.StageCrossplane}
	if c, ok := s.XR.reconcileError(); ok {
		v.Status, v.ActionOwner = domain.StageFailed, domain.OwnerPlatform
		v.Message, v.TechnicalDetail = "Erro ao compor a infraestrutura", c.Reason+": "+c.Message
		return v
	}
	if s.XR.synced() {
		v.Status, v.Message = domain.StageSucceeded, "Infraestrutura composta"
		return v
	}
	v.Status, v.ActionOwner, v.Message = domain.StageInProgress, domain.OwnerPlatform, "Compondo infraestrutura"
	return v
}

func cloudStage(s Signals) domain.StageView {
	v := domain.StageView{Stage: domain.StageCloud}
	for _, m := range s.MRs {
		if c, ok := m.reconcileError(); ok {
			v.Status, v.ActionOwner = domain.StageFailed, domain.OwnerPlatform
			v.Message, v.TechnicalDetail = "Erro ao provisionar na cloud", c.Reason+": "+c.Message
			return v
		}
	}
	if s.mrsReady() {
		v.Status, v.Message = domain.StageSucceeded, "Recursos de cloud prontos"
		return v
	}
	ready := 0
	for _, m := range s.MRs {
		if m.ready() {
			ready++
		}
	}
	v.Status, v.ActionOwner = domain.StageInProgress, domain.OwnerPlatform
	v.Message = fmt.Sprintf("Provisionando na cloud (%d/%d prontos)", ready, len(s.MRs))
	return v
}

func operationalStage(s Signals) domain.StageView {
	v := domain.StageView{Stage: domain.StageOperational}
	missing := ""
	switch {
	case !s.Claim.ready():
		missing = "Claim não está Ready"
	case !s.XR.ready():
		missing = "XR não está Ready"
	case !s.mrsReady():
		missing = "Managed Resources não estão Ready"
	case !s.SecretAvailable:
		missing = "Secret de conexão indisponível"
	case s.CatalogRegistered != nil && !*s.CatalogRegistered:
		missing = "Recurso não registrado no catálogo"
	}
	if missing == "" {
		v.Status, v.Message = domain.StageSucceeded, "Recurso operacional"
		return v
	}
	v.Status, v.ActionOwner, v.Message, v.TechnicalDetail = domain.StageInProgress, domain.OwnerPlatform, "Finalizando", missing
	return v
}
