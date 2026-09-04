package application

import (
	"context"
	"github.com/buffo/arch-view/internal/okf/domain"
)

// API is the operation boundary for a complete OKF transport. Construction and
// dependency configuration are deliberately excluded. Narrow clients can use
// the individual operation groups.
type API interface {
	CatalogOperations
	SessionOperations
	ProfileOperations
	InspectionOperations
	DiagnosticOperations
}

type DiagnosticOperations interface {
	Diagnostics(context.Context, DiagnosticQuery) (DiagnosticReport, error)
}

type CatalogOperations interface {
	Catalog() domain.BundleCatalog
	Refresh(context.Context) (domain.BundleCatalog, error)
	Summary(context.Context, string) (domain.BundleSummary, error)
	Profiles(context.Context) (domain.ProfileCatalog, error)
	Extensions(context.Context) ([]any, error)
}

type SessionOperations interface {
	Session(context.Context, string) (SessionView, error)
	Projection(context.Context, string) (SessionView, error)
	SelectBundle(context.Context, string, string) (SessionView, error)
	SelectProfile(context.Context, string, string) (SessionView, error)
	SetNavigation(context.Context, string, int, bool) (SessionView, error)
	Focus(context.Context, string, string) (SessionView, error)
	Back(context.Context, string) (SessionView, error)
	TopLevel(context.Context, string) (SessionView, error)
}

type InspectionOperations interface {
	Detail(context.Context, string, string) (domain.ConceptDetail, error)
}

type ProfileOperations interface {
	PreviewProfile(context.Context, domain.Profile) (ProfilePreview, error)
	ValidateProfile(context.Context, domain.Profile) (domain.Profile, []domain.Diagnostic)
	Bind(context.Context, string, string, string, string, []byte) (domain.ProjectConfiguration, error)
	SaveProfile(context.Context, domain.Profile, string, string, []byte) (domain.ProjectConfiguration, error)
	SaveProfileAs(context.Context, domain.Profile, string, string, string, string, []byte) (domain.ProjectConfiguration, error)
	RenameProfile(context.Context, string, string, string, string, string, []byte) (domain.ProjectConfiguration, error)
	DeleteProfile(context.Context, string, string, bool, string, string, []byte) (domain.ProjectConfiguration, error)
}

var _ API = (*Service)(nil)
