package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
)

// PGRepository provides PostgreSQL-backed storage for feedback data.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed feedback repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// Submit persists a feedback entry.
func (r *PGRepository) Submit(ctx context.Context, fb *Feedback) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, fb.TenantID, `
		INSERT INTO feedback (id, tenant_id, agent_id, span_id, task_id, rating, comment, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		fb.ID, fb.TenantID, fb.AgentID, fb.SpanID, fb.TaskID, fb.Rating, fb.Comment, fb.UserID, fb.CreatedAt)
}

// List returns all feedback for a tenant, optionally filtered by agent.
func (r *PGRepository) List(ctx context.Context, tenantID, agentID string) ([]*Feedback, error) {
	query := `SELECT id, tenant_id, agent_id, span_id, task_id, rating, comment, user_id, created_at
		FROM feedback WHERE tenant_id = $1`
	args := []any{tenantID}

	if agentID != "" {
		query += " AND agent_id = $2"
		args = append(args, agentID)
	}
	query += " ORDER BY created_at DESC"

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var feedbacks []*Feedback
	for rows.Next() {
		var fb Feedback
		if err := rows.Scan(&fb.ID, &fb.TenantID, &fb.AgentID, &fb.SpanID, &fb.TaskID, &fb.Rating, &fb.Comment, &fb.UserID, &fb.CreatedAt); err != nil {
			return nil, err
		}
		feedbacks = append(feedbacks, &fb)
	}
	return feedbacks, rows.Err()
}

// GetSummary returns aggregated feedback summaries by agent for a tenant.
func (r *PGRepository) GetSummary(ctx context.Context, tenantID string) ([]*FeedbackSummary, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT agent_id,
			COUNT(*) AS total_feedback,
			AVG(CASE WHEN rating > 0 THEN 1.0 ELSE 0.0 END) AS average_rating,
			COUNT(*) FILTER (WHERE rating > 0) AS positive_count,
			COUNT(*) FILTER (WHERE rating <= 0) AS negative_count
		FROM feedback WHERE tenant_id = $1
		GROUP BY agent_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var summaries []*FeedbackSummary
	for rows.Next() {
		var s FeedbackSummary
		if err := rows.Scan(&s.AgentID, &s.TotalFeedback, &s.AverageRating, &s.PositiveCount, &s.NegativeCount); err != nil {
			return nil, err
		}
		summaries = append(summaries, &s)
	}
	return summaries, rows.Err()
}

// GenerateID creates a unique feedback ID.
func GenerateID() string {
	return "fb-" + time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", time.Now().UnixNano()%10000)
}
