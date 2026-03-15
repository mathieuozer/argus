package dashboard

// Handler provides REST/gRPC handlers for the dashboard frontend.
type Handler struct{}

// New creates a new dashboard handler.
func New() *Handler {
	return &Handler{}
}
