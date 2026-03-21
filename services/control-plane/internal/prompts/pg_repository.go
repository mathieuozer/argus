package prompts

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGRepository provides PostgreSQL-backed storage for prompt data.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed prompts repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// CreatePrompt persists a new prompt.
func (r *PGRepository) CreatePrompt(ctx context.Context, prompt *Prompt) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, prompt.TenantID, `
		INSERT INTO prompts (id, tenant_id, name, description, agent_id, active_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		prompt.ID, prompt.TenantID, prompt.Name, prompt.Description, prompt.AgentID, prompt.ActiveVersion, prompt.CreatedAt, prompt.UpdatedAt)
}

// GetPrompt retrieves a prompt by ID.
func (r *PGRepository) GetPrompt(ctx context.Context, tenantID, promptID string) (*Prompt, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, agent_id, active_version, created_at, updated_at
		FROM prompts WHERE tenant_id = $1 AND id = $2`, tenantID, promptID)

	prompt, err := scanPromptRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return prompt, nil
}

// ListPrompts returns all prompts for a tenant.
func (r *PGRepository) ListPrompts(ctx context.Context, tenantID string) ([]*Prompt, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, name, description, agent_id, active_version, created_at, updated_at
		FROM prompts WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var prompts []*Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.AgentID, &p.ActiveVersion, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prompts = append(prompts, &p)
	}
	return prompts, rows.Err()
}

// CreateVersion persists a new prompt version.
func (r *PGRepository) CreateVersion(ctx context.Context, version *PromptVersion) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, version.TenantID, `
		INSERT INTO prompt_versions (id, prompt_id, tenant_id, version, content, change_log, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		version.ID, version.PromptID, version.TenantID, version.Version, version.Content, version.ChangeLog, version.CreatedAt)
}

// ListVersions returns all versions for a prompt.
func (r *PGRepository) ListVersions(ctx context.Context, tenantID, promptID string) ([]*PromptVersion, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, prompt_id, tenant_id, version, content, change_log, created_at
		FROM prompt_versions WHERE tenant_id = $1 AND prompt_id = $2
		ORDER BY version ASC`, tenantID, promptID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var versions []*PromptVersion
	for rows.Next() {
		var v PromptVersion
		if err := rows.Scan(&v.ID, &v.PromptID, &v.TenantID, &v.Version, &v.Content, &v.ChangeLog, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, &v)
	}
	return versions, rows.Err()
}

// CountVersions returns the count of versions for a prompt.
func (r *PGRepository) CountVersions(ctx context.Context, tenantID, promptID string) (int, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	var count int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM prompt_versions WHERE tenant_id = $1 AND prompt_id = $2`,
		tenantID, promptID).Scan(&count)
	return count, err
}

// SetActiveVersion updates the active version for a prompt.
func (r *PGRepository) SetActiveVersion(ctx context.Context, tenantID, promptID string, version int) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, tenantID, `
		UPDATE prompts SET active_version = $1, updated_at = $2
		WHERE tenant_id = $3 AND id = $4`,
		version, time.Now(), tenantID, promptID)
}

// GenerateID creates a unique prompt ID.
func GenerateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", time.Now().UnixNano()%10000)
}

func scanPromptRow(row pgx.Row) (*Prompt, error) {
	var p Prompt
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.AgentID, &p.ActiveVersion, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}
