package lifecycle

import "github.com/agnaldom/crosswatc/internal/domain"

// DeriveResourceState applies the ordered rules of spec §12.
func DeriveResourceState(s Signals) domain.ResourceState {
	open := s.OpenRevision
	updating := open != nil && (open.Kind == domain.RevisionUpdate || open.Kind == domain.RevisionPlatformUpgrade)

	switch {
	case s.Claim.Present && s.Claim.Deleting:
		return domain.ResourceDeleting
	case open != nil && open.Kind == domain.RevisionDelete:
		return domain.ResourceDeletePending
	case !s.Claim.Present && s.DeleteSucceeded:
		return domain.ResourceDeleted
	case !s.Claim.Present, s.InvalidMetadata:
		return domain.ResourceUnknown
	case !s.EverOperational && s.CreateFailed:
		return domain.ResourceFailed
	case s.AllHealthy() && updating:
		return domain.ResourceUpdating
	case s.AllHealthy():
		return domain.ResourceOperational
	case s.EverOperational && updating:
		return domain.ResourceUpdating
	case s.EverOperational:
		return domain.ResourceDegraded
	default:
		return domain.ResourceProvisioning
	}
}
