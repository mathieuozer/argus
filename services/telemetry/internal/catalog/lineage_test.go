package catalog

import (
	"testing"
)

func TestNewLineageBuilder(t *testing.T) {
	d := NewDiscoverer()
	lb := NewLineageBuilder(d)
	if lb == nil {
		t.Fatal("NewLineageBuilder() returned nil")
	}
}

func TestLineageBuilder_BuildGraph_Empty(t *testing.T) {
	d := NewDiscoverer()
	lb := NewLineageBuilder(d)

	graph := lb.BuildGraph("tenant-1")
	if graph == nil {
		t.Fatal("BuildGraph returned nil")
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(graph.Edges))
	}
}

func TestLineageBuilder_BuildGraph_SingleReadSource(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.read", map[string]string{
		"db.name": "analytics_db",
	})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes (source + agent), got %d", len(graph.Nodes))
	}

	// Should have a read edge: source -> agent
	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Edges))
	}

	edge := graph.Edges[0]
	if edge.Label != "read" {
		t.Errorf("edge label: got %q, want %q", edge.Label, "read")
	}
	if edge.Source != "analytics_db" {
		t.Errorf("edge source: got %q, want %q", edge.Source, "analytics_db")
	}
	if edge.Target != "agent-1" {
		t.Errorf("edge target: got %q, want %q", edge.Target, "agent-1")
	}
}

func TestLineageBuilder_BuildGraph_WriteSource(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.insert", map[string]string{
		"db.name": "output_db",
	})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Edges))
	}

	edge := graph.Edges[0]
	if edge.Label != "write" {
		t.Errorf("edge label: got %q, want %q", edge.Label, "write")
	}
	// Write edge: agent -> source
	if edge.Source != "agent-1" {
		t.Errorf("edge source: got %q, want %q", edge.Source, "agent-1")
	}
	if edge.Target != "output_db" {
		t.Errorf("edge target: got %q, want %q", edge.Target, "output_db")
	}
}

func TestLineageBuilder_BuildGraph_CallSource(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "llm.completion", map[string]string{
		"model": "gpt-4",
	})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Edges))
	}

	edge := graph.Edges[0]
	if edge.Label != "call" {
		t.Errorf("edge label: got %q, want %q", edge.Label, "call")
	}
	// Call edge: agent -> source
	if edge.Source != "agent-1" {
		t.Errorf("edge source: got %q, want %q", edge.Source, "agent-1")
	}
}

func TestLineageBuilder_BuildGraph_APINodeType(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "http.get", map[string]string{
		"http.url": "https://api.example.com/data",
	})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	foundExternalAPI := false
	for _, node := range graph.Nodes {
		if node.Type == "external_api" {
			foundExternalAPI = true
			break
		}
	}
	if !foundExternalAPI {
		t.Error("expected a node with type 'external_api' for API sources")
	}
}

func TestLineageBuilder_BuildGraph_MultipleAgentsSameSource(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.read", map[string]string{"db.name": "shared_db"})
	d.DiscoverFromSpan("tenant-1", "agent-2", "db.query.read", map[string]string{"db.name": "shared_db"})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	// Should have 3 nodes: shared_db, agent-1, agent-2
	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(graph.Nodes))
	}

	// Should have 2 read edges (one per agent)
	readEdges := 0
	for _, edge := range graph.Edges {
		if edge.Label == "read" {
			readEdges++
		}
	}
	if readEdges != 2 {
		t.Errorf("expected 2 read edges, got %d", readEdges)
	}
}

func TestLineageBuilder_BuildGraph_MultipleAccessTypes(t *testing.T) {
	d := NewDiscoverer()
	// First: read
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.read", map[string]string{"db.name": "mydb"})
	// Second: write (same source)
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.insert.write", map[string]string{"db.name": "mydb"})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	// Should have 2 nodes: mydb + agent-1
	if len(graph.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(graph.Nodes))
	}

	// Should have 2 edges: one read, one write
	if len(graph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(graph.Edges))
	}

	labels := make(map[string]bool)
	for _, edge := range graph.Edges {
		labels[edge.Label] = true
	}
	if !labels["read"] {
		t.Error("expected a 'read' edge")
	}
	if !labels["write"] {
		t.Error("expected a 'write' edge")
	}
}

func TestLineageBuilder_BuildGraph_TenantIsolation(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query", map[string]string{"db.name": "db1"})
	d.DiscoverFromSpan("tenant-2", "agent-2", "db.query", map[string]string{"db.name": "db2"})

	lb := NewLineageBuilder(d)

	graph1 := lb.BuildGraph("tenant-1")
	graph2 := lb.BuildGraph("tenant-2")

	if len(graph1.Nodes) != 2 {
		t.Errorf("tenant-1 nodes: got %d, want 2", len(graph1.Nodes))
	}
	if len(graph2.Nodes) != 2 {
		t.Errorf("tenant-2 nodes: got %d, want 2", len(graph2.Nodes))
	}

	// Verify no cross-tenant leakage
	for _, node := range graph1.Nodes {
		if node.ID == "agent-2" || node.ID == "db2" {
			t.Errorf("tenant-1 graph should not contain tenant-2 data, found node %q", node.ID)
		}
	}
}

func TestLineageBuilder_BuildGraph_SpanCount(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.read", map[string]string{"db.name": "mydb"})
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.read", map[string]string{"db.name": "mydb"})
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.read", map[string]string{"db.name": "mydb"})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Edges))
	}

	if graph.Edges[0].SpanCount != 3 {
		t.Errorf("edge SpanCount: got %d, want 3", graph.Edges[0].SpanCount)
	}
}

func TestLineageBuilder_BuildGraph_DatabaseNodeType(t *testing.T) {
	d := NewDiscoverer()
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query", map[string]string{"db.name": "mydb"})

	lb := NewLineageBuilder(d)
	graph := lb.BuildGraph("tenant-1")

	foundDB := false
	foundAgent := false
	for _, node := range graph.Nodes {
		switch node.Type {
		case "database":
			foundDB = true
		case "agent":
			foundAgent = true
		}
	}
	if !foundDB {
		t.Error("expected a node with type 'database'")
	}
	if !foundAgent {
		t.Error("expected a node with type 'agent'")
	}
}
