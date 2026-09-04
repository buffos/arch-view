package viewer

import "github.com/buffo/arch-view/internal/okf/application"

// Default local composition installs the renderer validator before publishing
// the application boundary. Injected applications keep their own dependencies.
func newOKFApplication(root string) application.API {
	service := application.New(root)
	service.SetLayoutValidator(okfLayoutValidator{})
	return service
}
