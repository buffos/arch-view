package application

// Service composes the local application boundary. Its promoted methods retain
// the transport API while each use-case group has a separate implementation.
type Service struct {
	*applicationState
	*catalogService
	*sessionService
	*inspectionService
	*profileService
	*diagnosticService
}

// These services share one publication lock and configuration-write lock because
// refresh and profile persistence must invalidate sessions atomically. They do
// not call through the facade or depend on unrelated use-case services.
type catalogService struct{ *applicationState }
type sessionService struct{ *applicationState }
type inspectionService struct{ *applicationState }
type profileService struct{ *applicationState }
type diagnosticService struct{ *applicationState }

func composeService(state *applicationState) *Service {
	catalog := &catalogService{state}
	state.initialize = catalog.Refresh
	return &Service{
		applicationState:  state,
		catalogService:    catalog,
		sessionService:    &sessionService{state},
		inspectionService: &inspectionService{state},
		profileService:    &profileService{state},
		diagnosticService: &diagnosticService{state},
	}
}

var (
	_ SessionOperations    = (*sessionService)(nil)
	_ InspectionOperations = (*inspectionService)(nil)
	_ ProfileOperations    = (*profileService)(nil)
	_ DiagnosticOperations = (*diagnosticService)(nil)
)
