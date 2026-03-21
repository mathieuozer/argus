package catalog

type LineageNode struct {
	ID   string `json:"id"`
	Type string `json:"type"` // agent, database, storage, external_api, tool
}

type LineageEdge struct {
	Source    string `json:"source"`
	Target   string `json:"target"`
	Label    string `json:"label"`
	SpanCount int64  `json:"span_count"`
}

type LineageGraph struct {
	Nodes []LineageNode `json:"nodes"`
	Edges []LineageEdge `json:"edges"`
}

type LineageBuilder struct {
	discoverer *Discoverer
}

func NewLineageBuilder(discoverer *Discoverer) *LineageBuilder {
	return &LineageBuilder{discoverer: discoverer}
}

func (lb *LineageBuilder) BuildGraph(tenantID string) *LineageGraph {
	sources := lb.discoverer.ListSources(tenantID)

	nodeSet := make(map[string]LineageNode)
	var edges []LineageEdge

	for _, source := range sources {
		// Add source node
		nodeType := string(source.Type)
		if source.Type == SourceAPI {
			nodeType = "external_api"
		}
		nodeSet[source.Identifier] = LineageNode{
			ID:   source.Name,
			Type: nodeType,
		}

		// Add agent nodes and edges
		for _, agentID := range source.Agents {
			nodeSet[agentID] = LineageNode{
				ID:   agentID,
				Type: "agent",
			}

			for _, accessType := range source.AccessTypes {
				switch accessType {
				case "read":
					edges = append(edges, LineageEdge{
						Source:    source.Name,
						Target:   agentID,
						Label:    "read",
						SpanCount: source.SpanCount,
					})
				case "write":
					edges = append(edges, LineageEdge{
						Source:    agentID,
						Target:   source.Name,
						Label:    "write",
						SpanCount: source.SpanCount,
					})
				case "call":
					edges = append(edges, LineageEdge{
						Source:    agentID,
						Target:   source.Name,
						Label:    "call",
						SpanCount: source.SpanCount,
					})
				}
			}
		}
	}

	var nodes []LineageNode
	for _, node := range nodeSet {
		nodes = append(nodes, node)
	}

	return &LineageGraph{Nodes: nodes, Edges: edges}
}
