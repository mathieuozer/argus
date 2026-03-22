package feedback

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

type Feedback struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	AgentID   string    `json:"agent_id"`
	SpanID    string    `json:"span_id"`
	TaskID    string    `json:"task_id"`
	Rating    int       `json:"rating"` // 1-5 or -1/1 for thumbs
	Comment   string    `json:"comment"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type FeedbackSummary struct {
	AgentID       string  `json:"agent_id"`
	TotalFeedback int     `json:"total_feedback"`
	AverageRating float64 `json:"average_rating"`
	PositiveCount int     `json:"positive_count"`
	NegativeCount int     `json:"negative_count"`
}

type Repository struct {
	mu       sync.RWMutex
	feedback []*Feedback
}

func NewRepository() *Repository {
	return &Repository{feedback: make([]*Feedback, 0)}
}

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/feedback", h.SubmitFeedback)
	mux.HandleFunc("GET /api/v1/feedback", h.ListFeedback)
	mux.HandleFunc("GET /api/v1/feedback/summary", h.GetSummary)
}

func (h *Handler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	var fb Feedback
	if err := json.NewDecoder(r.Body).Decode(&fb); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	fb.ID = "fb-" + time.Now().Format("20060102150405")
	fb.TenantID = tenantID
	fb.CreatedAt = time.Now()

	h.repo.mu.Lock()
	h.repo.feedback = append(h.repo.feedback, &fb)
	h.repo.mu.Unlock()

	httputil.WriteJSON(w, http.StatusCreated, fb, "")
}

func (h *Handler) ListFeedback(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}
	agentID := r.URL.Query().Get("agent_id")

	h.repo.mu.RLock()
	var feedbacks []*Feedback
	for _, fb := range h.repo.feedback {
		if fb.TenantID == tenantID {
			if agentID == "" || fb.AgentID == agentID {
				feedbacks = append(feedbacks, fb)
			}
		}
	}
	h.repo.mu.RUnlock()

	if feedbacks == nil {
		feedbacks = []*Feedback{}
	}
	httputil.WriteJSON(w, http.StatusOK, feedbacks, "")
}

func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	h.repo.mu.RLock()
	summaries := make(map[string]*FeedbackSummary)
	for _, fb := range h.repo.feedback {
		if fb.TenantID != tenantID {
			continue
		}
		s, ok := summaries[fb.AgentID]
		if !ok {
			s = &FeedbackSummary{AgentID: fb.AgentID}
			summaries[fb.AgentID] = s
		}
		s.TotalFeedback++
		if fb.Rating > 0 {
			s.PositiveCount++
		} else {
			s.NegativeCount++
		}
		s.AverageRating = float64(s.PositiveCount) / float64(s.TotalFeedback)
	}
	h.repo.mu.RUnlock()

	var result []*FeedbackSummary
	for _, s := range summaries {
		result = append(result, s)
	}
	if result == nil {
		result = []*FeedbackSummary{}
	}
	httputil.WriteJSON(w, http.StatusOK, result, "")
}
