package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// Discovery handles auto-registration with the orchestrator.
type Discovery struct {
	logger           *zap.Logger
	orchestratorAddr string
	tenantID         string
	registered       bool
	agentID          string
}

// New creates a new discovery component.
func New(logger *zap.Logger) *Discovery {
	addr := os.Getenv("ARGUS_ORCHESTRATOR_ADDR")
	if addr == "" {
		addr = "http://localhost:8082"
	}
	tenantID := os.Getenv("ARGUS_TENANT_ID")
	if tenantID == "" {
		tenantID = "default"
	}

	return &Discovery{
		logger:           logger,
		orchestratorAddr: addr,
		tenantID:         tenantID,
	}
}

// Register registers this sidecar's agent with the orchestrator.
func (d *Discovery) Register(agentID, version, framework string, capabilities []string) error {
	d.logger.Info("registering agent",
		zap.String("agent_id", agentID),
		zap.String("version", version),
		zap.String("framework", framework),
		zap.String("orchestrator", d.orchestratorAddr),
	)

	hostname, _ := os.Hostname()
	body, _ := json.Marshal(map[string]interface{}{
		"agent_id":     agentID,
		"version":      version,
		"framework":    framework,
		"capabilities": capabilities,
		"node_id":      hostname,
	})

	req, err := http.NewRequest(http.MethodPost, d.orchestratorAddr+"/api/v1/agents", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", d.tenantID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		d.logger.Warn("failed to register with orchestrator, will retry", zap.Error(err))
		return fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	d.registered = true
	d.agentID = agentID
	d.logger.Info("agent registered successfully", zap.String("agent_id", agentID))
	return nil
}

// SendHeartbeat sends a heartbeat to the orchestrator.
func (d *Discovery) SendHeartbeat(status string) error {
	if !d.registered {
		return fmt.Errorf("agent not registered")
	}

	body, _ := json.Marshal(map[string]string{"status": status})
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/agents/%s/heartbeat", d.orchestratorAddr, d.agentID),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", d.tenantID)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// StartHeartbeatLoop starts a background heartbeat loop.
func (d *Discovery) StartHeartbeatLoop(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if d.registered {
				if err := d.SendHeartbeat("healthy"); err != nil {
					d.logger.Warn("heartbeat failed", zap.Error(err))
				}
			}
		}
	}()
}

// IsRegistered returns whether the agent has been registered.
func (d *Discovery) IsRegistered() bool {
	return d.registered
}

// AgentID returns the registered agent ID.
func (d *Discovery) AgentID() string {
	return d.agentID
}
