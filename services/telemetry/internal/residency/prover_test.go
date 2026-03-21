package residency

import (
	"testing"
)

func TestAttest(t *testing.T) {
	prover := NewProver("test-key")

	att := prover.Attest("tenant-1", "node-1", "sa-riyadh-1", []byte("sensitive data"))

	if att.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", att.TenantID)
	}
	if att.Region != "sa-riyadh-1" {
		t.Errorf("expected sa-riyadh-1, got %s", att.Region)
	}
	if att.DataHash == "" {
		t.Error("expected non-empty data hash")
	}
	if att.Signature == "" {
		t.Error("expected non-empty signature")
	}
}

func TestBuildMerkleRoot(t *testing.T) {
	prover := NewProver("test-key")

	prover.Attest("tenant-1", "node-1", "sa-riyadh-1", []byte("data-1"))
	prover.Attest("tenant-1", "node-1", "sa-riyadh-1", []byte("data-2"))
	prover.Attest("tenant-1", "node-2", "sa-jeddah-1", []byte("data-3"))

	root := prover.BuildMerkleRoot("tenant-1")
	if root == "" {
		t.Error("expected non-empty merkle root")
	}

	// Same data should produce same root
	root2 := prover.BuildMerkleRoot("tenant-1")
	if root != root2 {
		t.Error("merkle root should be deterministic")
	}
}

func TestBuildMerkleRoot_Empty(t *testing.T) {
	prover := NewProver("test-key")
	root := prover.BuildMerkleRoot("nonexistent")
	if root != "" {
		t.Error("expected empty root for no attestations")
	}
}

func TestGenerateProof_Compliant(t *testing.T) {
	prover := NewProver("test-key")

	prover.Attest("tenant-1", "node-1", "sa-riyadh-1", []byte("data-1"))
	prover.Attest("tenant-1", "node-2", "sa-jeddah-1", []byte("data-2"))

	proof := prover.GenerateProof("tenant-1", []string{"sa-riyadh-1", "sa-jeddah-1"})

	if !proof.Compliant {
		t.Error("expected compliant proof")
	}
	if len(proof.Attestations) != 2 {
		t.Errorf("expected 2 attestations, got %d", len(proof.Attestations))
	}
	if proof.MerkleRoot == "" {
		t.Error("expected non-empty merkle root")
	}
}

func TestGenerateProof_NonCompliant(t *testing.T) {
	prover := NewProver("test-key")

	prover.Attest("tenant-1", "node-1", "sa-riyadh-1", []byte("data-1"))
	prover.Attest("tenant-1", "node-2", "eu-west-1", []byte("data-2")) // Wrong region!

	proof := prover.GenerateProof("tenant-1", []string{"sa-riyadh-1", "sa-jeddah-1"})

	if proof.Compliant {
		t.Error("expected non-compliant proof (data in eu-west-1)")
	}
}

func TestTenantIsolation(t *testing.T) {
	prover := NewProver("test-key")

	prover.Attest("tenant-a", "node-1", "sa-riyadh-1", []byte("data-a"))
	prover.Attest("tenant-b", "node-2", "ae-dubai-1", []byte("data-b"))

	proofA := prover.GenerateProof("tenant-a", []string{"sa-riyadh-1"})
	proofB := prover.GenerateProof("tenant-b", []string{"ae-dubai-1"})

	if len(proofA.Attestations) != 1 {
		t.Errorf("expected 1 attestation for tenant-a, got %d", len(proofA.Attestations))
	}
	if len(proofB.Attestations) != 1 {
		t.Errorf("expected 1 attestation for tenant-b, got %d", len(proofB.Attestations))
	}
	if proofA.MerkleRoot == proofB.MerkleRoot {
		t.Error("merkle roots should differ between tenants")
	}
}

func TestBuildMerkleRoot_SingleAttestation(t *testing.T) {
	prover := NewProver("test-key")

	att := prover.Attest("tenant-1", "node-1", "sa-riyadh-1", []byte("single-data"))
	root := prover.BuildMerkleRoot("tenant-1")

	// With a single attestation, the merkle root should be the data hash itself
	if root != att.DataHash {
		t.Errorf("single attestation merkle root should equal data hash, got root=%s hash=%s", root, att.DataHash)
	}
}

func TestAttest_IDFormat(t *testing.T) {
	prover := NewProver("test-key")

	att := prover.Attest("short", "node-1", "region-1", []byte("data"))
	if att.ID == "" {
		t.Error("attestation ID should not be empty")
	}

	att2 := prover.Attest("a-very-long-tenant-id-that-exceeds-eight-chars", "node-1", "region-1", []byte("data"))
	if att2.ID == "" {
		t.Error("attestation ID should not be empty for long tenant IDs")
	}
}

func TestGenerateProof_NoAttestations(t *testing.T) {
	prover := NewProver("test-key")

	proof := prover.GenerateProof("empty-tenant", []string{"sa-riyadh-1"})

	if !proof.Compliant {
		t.Error("empty attestation set should be compliant (vacuously true)")
	}
	if proof.MerkleRoot != "" {
		t.Error("expected empty merkle root for no attestations")
	}
	if proof.Attestations != nil {
		t.Errorf("expected nil attestations, got %d", len(proof.Attestations))
	}
}
