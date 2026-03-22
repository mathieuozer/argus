package catalog

import (
	"testing"
)

func TestNewDiscoverer(t *testing.T) {
	d := NewDiscoverer()
	if d == nil {
		t.Fatal("NewDiscoverer() returned nil")
	}
	if d.entries == nil {
		t.Fatal("entries map not initialized")
	}
}

func TestDiscoverer_DiscoverFromSpan_Database(t *testing.T) {
	tests := []struct {
		name      string
		opName    string
		attrs     map[string]string
		wantType  SourceType
		wantName  string
		wantIdent string
		wantNil   bool
	}{
		{
			name:      "database via connection string",
			opName:    "db.query",
			attrs:     map[string]string{"db.connection_string": "postgres://host/mydb?sslmode=require"},
			wantType:  SourceDatabase,
			wantName:  "mydb",
			wantIdent: "postgres://host/mydb?sslmode=require",
		},
		{
			name:      "database via db.name",
			opName:    "sql.execute",
			attrs:     map[string]string{"db.name": "analytics"},
			wantType:  SourceDatabase,
			wantName:  "analytics",
			wantIdent: "analytics",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscoverer()
			entry := d.DiscoverFromSpan("tenant-1", "agent-1", tc.opName, tc.attrs)

			if tc.wantNil {
				if entry != nil {
					t.Errorf("expected nil entry, got %+v", entry)
				}
				return
			}

			if entry == nil {
				t.Fatal("expected non-nil entry")
			}
			if entry.Type != tc.wantType {
				t.Errorf("Type: got %q, want %q", entry.Type, tc.wantType)
			}
			if entry.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", entry.Name, tc.wantName)
			}
			if entry.Identifier != tc.wantIdent {
				t.Errorf("Identifier: got %q, want %q", entry.Identifier, tc.wantIdent)
			}
		})
	}
}

func TestDiscoverer_DiscoverFromSpan_API(t *testing.T) {
	tests := []struct {
		name      string
		opName    string
		attrs     map[string]string
		wantType  SourceType
		wantName  string
		wantIdent string
	}{
		{
			name:      "API via http.url",
			opName:    "http.request",
			attrs:     map[string]string{"http.url": "https://api.openai.com/v1/completions"},
			wantType:  SourceAPI,
			wantName:  "api.openai.com",
			wantIdent: "https://api.openai.com/v1/completions",
		},
		{
			name:      "LLM via model attribute",
			opName:    "llm.completion",
			attrs:     map[string]string{"model": "gpt-4"},
			wantType:  SourceAPI,
			wantName:  "gpt-4_api",
			wantIdent: "gpt-4",
		},
		{
			name:      "API call operation",
			opName:    "api.invoke",
			attrs:     map[string]string{"http.url": "http://internal:8080/api/v1/data"},
			wantType:  SourceAPI,
			wantName:  "internal",
			wantIdent: "http://internal:8080/api/v1/data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscoverer()
			entry := d.DiscoverFromSpan("tenant-1", "agent-1", tc.opName, tc.attrs)

			if entry == nil {
				t.Fatal("expected non-nil entry")
			}
			if entry.Type != tc.wantType {
				t.Errorf("Type: got %q, want %q", entry.Type, tc.wantType)
			}
			if entry.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", entry.Name, tc.wantName)
			}
			if entry.Identifier != tc.wantIdent {
				t.Errorf("Identifier: got %q, want %q", entry.Identifier, tc.wantIdent)
			}
		})
	}
}

func TestDiscoverer_DiscoverFromSpan_Storage(t *testing.T) {
	d := NewDiscoverer()
	entry := d.DiscoverFromSpan("tenant-1", "agent-1", "s3.upload", map[string]string{
		"storage.bucket": "telemetry-data",
	})

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Type != SourceStorage {
		t.Errorf("Type: got %q, want %q", entry.Type, SourceStorage)
	}
	if entry.Name != "telemetry-data" {
		t.Errorf("Name: got %q, want %q", entry.Name, "telemetry-data")
	}
}

func TestDiscoverer_DiscoverFromSpan_Tool(t *testing.T) {
	tests := []struct {
		name     string
		opName   string
		attrs    map[string]string
		wantName string
	}{
		{
			name:     "tool with tool.name attribute",
			opName:   "tool.execute",
			attrs:    map[string]string{"tool.name": "calculator"},
			wantName: "calculator",
		},
		{
			name:     "tool without tool.name uses operation name",
			opName:   "tool.run",
			attrs:    map[string]string{},
			wantName: "tool.run",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscoverer()
			entry := d.DiscoverFromSpan("tenant-1", "agent-1", tc.opName, tc.attrs)

			if entry == nil {
				t.Fatal("expected non-nil entry")
			}
			if entry.Type != SourceTool {
				t.Errorf("Type: got %q, want %q", entry.Type, SourceTool)
			}
			if entry.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", entry.Name, tc.wantName)
			}
		})
	}
}

func TestDiscoverer_DiscoverFromSpan_FallbackToTool(t *testing.T) {
	d := NewDiscoverer()
	entry := d.DiscoverFromSpan("tenant-1", "agent-1", "custom.operation", map[string]string{})

	if entry == nil {
		t.Fatal("expected non-nil entry for fallback")
	}
	if entry.Type != SourceTool {
		t.Errorf("Type: got %q, want %q (fallback)", entry.Type, SourceTool)
	}
	if entry.Name != "custom.operation" {
		t.Errorf("Name: got %q, want %q", entry.Name, "custom.operation")
	}
}

func TestDiscoverer_DiscoverFromSpan_UpdatesExisting(t *testing.T) {
	d := NewDiscoverer()

	// First discovery
	entry1 := d.DiscoverFromSpan("tenant-1", "agent-1", "db.query", map[string]string{
		"db.name": "mydb",
	})
	if entry1.SpanCount != 1 {
		t.Errorf("initial SpanCount: got %d, want 1", entry1.SpanCount)
	}

	// Second discovery by same agent
	entry2 := d.DiscoverFromSpan("tenant-1", "agent-1", "db.query", map[string]string{
		"db.name": "mydb",
	})
	if entry2.SpanCount != 2 {
		t.Errorf("updated SpanCount: got %d, want 2", entry2.SpanCount)
	}
	if len(entry2.Agents) != 1 {
		t.Errorf("agents should still be 1, got %d", len(entry2.Agents))
	}

	// Third discovery by different agent
	entry3 := d.DiscoverFromSpan("tenant-1", "agent-2", "db.select", map[string]string{
		"db.name": "mydb",
	})
	if entry3.SpanCount != 3 {
		t.Errorf("updated SpanCount: got %d, want 3", entry3.SpanCount)
	}
	if len(entry3.Agents) != 2 {
		t.Errorf("agents should be 2, got %d", len(entry3.Agents))
	}
}

func TestDiscoverer_DiscoverFromSpan_AccessTypes(t *testing.T) {
	d := NewDiscoverer()

	// Read operation
	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.read", map[string]string{
		"db.name": "mydb",
	})

	// Write operation to same source
	entry := d.DiscoverFromSpan("tenant-1", "agent-1", "db.query.write", map[string]string{
		"db.name": "mydb",
	})

	if len(entry.AccessTypes) != 2 {
		t.Fatalf("expected 2 access types, got %d: %v", len(entry.AccessTypes), entry.AccessTypes)
	}
	if !containsString(entry.AccessTypes, "read") {
		t.Error("missing 'read' access type")
	}
	if !containsString(entry.AccessTypes, "write") {
		t.Error("missing 'write' access type")
	}
}

func TestDiscoverer_ListSources(t *testing.T) {
	d := NewDiscoverer()

	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query", map[string]string{"db.name": "db1"})
	d.DiscoverFromSpan("tenant-1", "agent-2", "http.request", map[string]string{"http.url": "https://api.example.com"})
	d.DiscoverFromSpan("tenant-2", "agent-3", "db.query", map[string]string{"db.name": "db2"})

	tenant1Sources := d.ListSources("tenant-1")
	if len(tenant1Sources) != 2 {
		t.Errorf("tenant-1 sources: got %d, want 2", len(tenant1Sources))
	}

	tenant2Sources := d.ListSources("tenant-2")
	if len(tenant2Sources) != 1 {
		t.Errorf("tenant-2 sources: got %d, want 1", len(tenant2Sources))
	}

	noSources := d.ListSources("tenant-999")
	if len(noSources) != 0 {
		t.Errorf("non-existent tenant sources: got %d, want 0", len(noSources))
	}
}

func TestDiscoverer_ListSourcesByAgent(t *testing.T) {
	d := NewDiscoverer()

	d.DiscoverFromSpan("tenant-1", "agent-1", "db.query", map[string]string{"db.name": "db1"})
	d.DiscoverFromSpan("tenant-1", "agent-1", "http.request", map[string]string{"http.url": "https://api.example.com"})
	d.DiscoverFromSpan("tenant-1", "agent-2", "db.query", map[string]string{"db.name": "db2"})

	agent1Sources := d.ListSourcesByAgent("tenant-1", "agent-1")
	if len(agent1Sources) != 2 {
		t.Errorf("agent-1 sources: got %d, want 2", len(agent1Sources))
	}

	agent2Sources := d.ListSourcesByAgent("tenant-1", "agent-2")
	if len(agent2Sources) != 1 {
		t.Errorf("agent-2 sources: got %d, want 1", len(agent2Sources))
	}

	noSources := d.ListSourcesByAgent("tenant-1", "agent-999")
	if len(noSources) != 0 {
		t.Errorf("non-existent agent sources: got %d, want 0", len(noSources))
	}
}

func TestInferAccessType(t *testing.T) {
	tests := []struct {
		name   string
		opName string
		want   string
	}{
		{name: "read operation", opName: "db.read", want: "read"},
		{name: "get operation", opName: "http.get", want: "read"},
		{name: "query operation", opName: "db.query", want: "read"},
		{name: "select operation", opName: "sql.select", want: "read"},
		{name: "write operation", opName: "db.write", want: "write"},
		{name: "put operation", opName: "http.put", want: "write"},
		{name: "insert operation", opName: "sql.insert", want: "write"},
		{name: "create operation", opName: "api.create", want: "write"},
		{name: "call operation", opName: "tool.call", want: "call"},
		{name: "invoke operation", opName: "function.invoke", want: "call"},
		{name: "completion operation", opName: "llm.completion", want: "call"},
		{name: "unknown operation", opName: "custom.operation", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := inferAccessType(tc.opName)
			if got != tc.want {
				t.Errorf("inferAccessType(%q) = %q, want %q", tc.opName, got, tc.want)
			}
		})
	}
}

func TestExtractDBName(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{name: "postgres dsn", dsn: "postgres://user:pass@host:5432/mydb?sslmode=require", want: "mydb"},
		{name: "simple path", dsn: "host/dbname", want: "dbname"},
		{name: "no db name", dsn: "host/", want: "unknown_db"},
		{name: "just host", dsn: "host", want: "host"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractDBName(tc.dsn)
			if got != tc.want {
				t.Errorf("extractDBName(%q) = %q, want %q", tc.dsn, got, tc.want)
			}
		})
	}
}

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "https url", url: "https://api.example.com/v1/data", want: "api.example.com"},
		{name: "http url", url: "http://localhost:8080/api", want: "localhost"},
		{name: "no scheme", url: "example.com/path", want: "example.com"},
		{name: "with port", url: "https://host:9090/path", want: "host"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractHostname(tc.url)
			if got != tc.want {
				t.Errorf("extractHostname(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
		want  bool
	}{
		{name: "found", slice: []string{"a", "b", "c"}, s: "b", want: true},
		{name: "not found", slice: []string{"a", "b", "c"}, s: "d", want: false},
		{name: "empty slice", slice: []string{}, s: "a", want: false},
		{name: "nil slice", slice: nil, s: "a", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := containsString(tc.slice, tc.s)
			if got != tc.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", tc.slice, tc.s, got, tc.want)
			}
		})
	}
}

func TestDiscoverer_IDGeneration(t *testing.T) {
	d := NewDiscoverer()

	e1 := d.DiscoverFromSpan("t1", "a1", "db.query", map[string]string{"db.name": "db1"})
	e2 := d.DiscoverFromSpan("t1", "a1", "http.get", map[string]string{"http.url": "https://api.com"})

	if e1.ID == e2.ID {
		t.Errorf("entries should have unique IDs, both got %q", e1.ID)
	}
	if e1.ID != "src-001" {
		t.Errorf("first entry ID: got %q, want %q", e1.ID, "src-001")
	}
	if e2.ID != "src-002" {
		t.Errorf("second entry ID: got %q, want %q", e2.ID, "src-002")
	}
}

func TestDiscoverer_TenantIsolation(t *testing.T) {
	d := NewDiscoverer()

	// Same db.name but different tenants should be separate entries
	e1 := d.DiscoverFromSpan("tenant-1", "agent-1", "db.query", map[string]string{"db.name": "shared_db"})
	e2 := d.DiscoverFromSpan("tenant-2", "agent-2", "db.query", map[string]string{"db.name": "shared_db"})

	if e1.ID == e2.ID {
		t.Error("entries from different tenants should have different IDs")
	}

	tenant1Sources := d.ListSources("tenant-1")
	tenant2Sources := d.ListSources("tenant-2")

	if len(tenant1Sources) != 1 || len(tenant2Sources) != 1 {
		t.Errorf("each tenant should have exactly 1 source, got tenant-1=%d, tenant-2=%d",
			len(tenant1Sources), len(tenant2Sources))
	}
}
