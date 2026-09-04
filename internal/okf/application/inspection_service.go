package application

import (
	"context"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func (service *inspectionService) Detail(ctx context.Context, sessionID, conceptID string) (domain.ConceptDetail, error) {
	if err := service.ensureReady(ctx); err != nil {
		return domain.ConceptDetail{}, err
	}
	service.mu.RLock()
	session := service.sessionLockedRead(sessionID)
	index, exists := service.indexes[session.BundleID]
	service.mu.RUnlock()
	if !exists {
		return domain.ConceptDetail{}, domain.NewError("okf_bundle_not_found", 404, "the session bundle is not available", nil)
	}
	return (conceptInspector{profiles: service.registry}).inspect(ctx, index, session.ProfileID, conceptID)
}
