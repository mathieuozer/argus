package residency

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Attestation is a signed proof that data resides at a specific location.
type Attestation struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	NodeID    string    `json:"node_id"`
	Region    string    `json:"region"`
	DataHash  string    `json:"data_hash"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
}

// MerkleNode represents a node in the Merkle tree.
type MerkleNode struct {
	Hash  string      `json:"hash"`
	Left  *MerkleNode `json:"left,omitempty"`
	Right *MerkleNode `json:"right,omitempty"`
}

// ResidencyProof is the complete proof for a tenant's data residency.
type ResidencyProof struct {
	TenantID       string         `json:"tenant_id"`
	MerkleRoot     string         `json:"merkle_root"`
	Attestations   []*Attestation `json:"attestations"`
	AllowedRegions []string       `json:"allowed_regions"`
	Compliant      bool           `json:"compliant"`
	VerifiedAt     time.Time      `json:"verified_at"`
}

// Prover generates data residency attestations and Merkle proofs.
type Prover struct {
	mu           sync.RWMutex
	attestations map[string][]*Attestation // keyed by tenantID
	signingKey   string
}

// NewProver creates a new residency prover.
func NewProver(signingKey string) *Prover {
	return &Prover{
		attestations: make(map[string][]*Attestation),
		signingKey:   signingKey,
	}
}

// Attest creates a signed attestation for a data write.
func (p *Prover) Attest(tenantID, nodeID, region string, data []byte) *Attestation {
	dataHash := hashBytes(data)
	attestation := &Attestation{
		ID:        fmt.Sprintf("att-%s-%d", tenantID[:minInt(8, len(tenantID))], time.Now().UnixNano()),
		TenantID:  tenantID,
		NodeID:    nodeID,
		Region:    region,
		DataHash:  dataHash,
		Timestamp: time.Now(),
		Signature: p.sign(tenantID + nodeID + region + dataHash),
	}

	p.mu.Lock()
	p.attestations[tenantID] = append(p.attestations[tenantID], attestation)
	p.mu.Unlock()

	return attestation
}

// BuildMerkleRoot builds a Merkle root from all attestations for a tenant.
func (p *Prover) BuildMerkleRoot(tenantID string) string {
	p.mu.RLock()
	atts := p.attestations[tenantID]
	p.mu.RUnlock()

	if len(atts) == 0 {
		return ""
	}

	var hashes []string
	for _, att := range atts {
		hashes = append(hashes, att.DataHash)
	}

	return computeMerkleRoot(hashes)
}

// GenerateProof creates a full residency proof for a tenant.
func (p *Prover) GenerateProof(tenantID string, allowedRegions []string) *ResidencyProof {
	p.mu.RLock()
	atts := p.attestations[tenantID]
	p.mu.RUnlock()

	proof := &ResidencyProof{
		TenantID:       tenantID,
		MerkleRoot:     p.BuildMerkleRoot(tenantID),
		Attestations:   atts,
		AllowedRegions: allowedRegions,
		Compliant:      true,
		VerifiedAt:     time.Now(),
	}

	// Check all attestations are in allowed regions
	for _, att := range atts {
		if !containsStr(allowedRegions, att.Region) {
			proof.Compliant = false
			break
		}
	}

	return proof
}

func (p *Prover) sign(data string) string {
	h := sha256.New()
	h.Write([]byte(data + p.signingKey))
	return hex.EncodeToString(h.Sum(nil))
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func computeMerkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return ""
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	var nextLevel []string
	for i := 0; i < len(hashes); i += 2 {
		if i+1 < len(hashes) {
			combined := hashes[i] + hashes[i+1]
			h := sha256.Sum256([]byte(combined))
			nextLevel = append(nextLevel, hex.EncodeToString(h[:]))
		} else {
			nextLevel = append(nextLevel, hashes[i])
		}
	}
	return computeMerkleRoot(nextLevel)
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
