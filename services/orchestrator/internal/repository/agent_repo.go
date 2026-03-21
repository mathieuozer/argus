package repository

import (
	"context"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/argus-platform/argus/pkg/errors"
	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"github.com/jackc/pgx/v5"
)

// AgentRepository provides PostgreSQL-backed agent persistence.
type AgentRepository struct {
	pool *database.Pool
}

// NewAgentRepository creates a new PostgreSQL-backed agent repository.
func NewAgentRepository(pool *database.Pool) *AgentRepository {
	return &AgentRepository{pool: pool}
}

// Register persists a new agent or updates an existing one.
func (r *AgentRepository) Register(ctx context.Context, tenantID string, req *registry.RegisterRequest) (*registry.Agent, error) {
	capabilities := req.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}

	agent := &registry.Agent{
		ID:           req.AgentID,
		TenantID:     tenantID,
		Version:      req.Version,
		Framework:    req.Framework,
		Capabilities: capabilities,
		Status:       registry.StatusDiscovered,
		LastSeen:     time.Now(),
		NodeID:       req.NodeID,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO agents (id, tenant_id, version, framework, capabilities, status, last_seen, node_id)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, id) DO UPDATE SET
			version = EXCLUDED.version,
			framework = EXCLUDED.framework,
			capabilities = EXCLUDED.capabilities,
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen,
			node_id = EXCLUDED.node_id`,
		agent.ID, tenantID, agent.Version, agent.Framework, agent.Capabilities,
		string(agent.Status), agent.LastSeen, agent.NodeID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}

	return agent, tx.Commit(ctx)
}

// Get retrieves an agent by tenant and agent ID.
func (r *AgentRepository) Get(ctx context.Context, tenantID, agentID string) (*registry.Agent, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, version, framework, capabilities, status, svid_uri, last_seen, node_id
		FROM agents WHERE tenant_id = $1::uuid AND id = $2`, tenantID, agentID)

	agent, err := scanAgent(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.CodeAgentNotFound, "agent not found")
		}
		return nil, err
	}
	return agent, nil
}

// List returns all agents for a tenant.
func (r *AgentRepository) List(ctx context.Context, tenantID string) ([]*registry.Agent, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, version, framework, capabilities, status, svid_uri, last_seen, node_id
		FROM agents WHERE tenant_id = $1::uuid
		ORDER BY last_seen DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var agents []*registry.Agent
	for rows.Next() {
		agent, err := scanAgentRows(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

// Heartbeat updates agent status and last_seen.
func (r *AgentRepository) Heartbeat(ctx context.Context, tenantID, agentID string, status registry.AgentStatus) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, tenantID, `
		UPDATE agents SET status = $1, last_seen = $2
		WHERE tenant_id = $3::uuid AND id = $4`,
		string(status), time.Now(), tenantID, agentID)
}

// Quarantine sets an agent's status to quarantined.
func (r *AgentRepository) Quarantine(ctx context.Context, tenantID, agentID string) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, tenantID, `
		UPDATE agents SET status = 'quarantined', last_seen = $1
		WHERE tenant_id = $2::uuid AND id = $3`,
		time.Now(), tenantID, agentID)
}

// FindByCapabilities returns agents with matching capabilities.
func (r *AgentRepository) FindByCapabilities(ctx context.Context, tenantID string, capabilities []string) ([]*registry.Agent, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, version, framework, capabilities, status, svid_uri, last_seen, node_id
		FROM agents
		WHERE tenant_id = $1::uuid
		  AND status NOT IN ('quarantined', 'failed')
		  AND capabilities @> $2
		ORDER BY last_seen DESC`, tenantID, capabilities)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var agents []*registry.Agent
	for rows.Next() {
		agent, err := scanAgentRows(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func scanAgent(row pgx.Row) (*registry.Agent, error) {
	var a registry.Agent
	var svidURI *string
	err := row.Scan(&a.ID, &a.TenantID, &a.Version, &a.Framework, &a.Capabilities,
		&a.Status, &svidURI, &a.LastSeen, &a.NodeID)
	if err != nil {
		return nil, err
	}
	if svidURI != nil {
		a.SVIDURI = *svidURI
	}
	return &a, nil
}

func scanAgentRows(rows pgx.Rows) (*registry.Agent, error) {
	var a registry.Agent
	var svidURI *string
	err := rows.Scan(&a.ID, &a.TenantID, &a.Version, &a.Framework, &a.Capabilities,
		&a.Status, &svidURI, &a.LastSeen, &a.NodeID)
	if err != nil {
		return nil, err
	}
	if svidURI != nil {
		a.SVIDURI = *svidURI
	}
	return &a, nil
}
