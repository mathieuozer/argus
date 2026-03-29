export interface Column {
  name: string;
  type: string;
  description: string;
  is_pii: boolean;
  is_nullable: boolean;
  classification: string;
  tags: string[];
  sample_values?: string[];
  null_rate: number;
  unique_count: number;
  min_value?: string;
  max_value?: string;
}

export interface FreshnessInfo {
  last_refreshed: string;
  refresh_frequency: string;
  sla_seconds: number;
  is_stale: boolean;
  stale_since?: string;
}

export interface PopularityInfo {
  view_count: number;
  query_count: number;
  unique_users: number;
  trend_direction: string;
  popularity_rank: number;
}

export interface ProfileInfo {
  row_count: number;
  column_count: number;
  size_bytes: number;
  null_rate: number;
  duplicate_rate: number;
  completeness: number;
  last_profiled: string;
}

export interface CatalogSource {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: 'database' | 'api' | 'file' | 'stream' | 'model' | 'storage' | 'tool';
  owner: string;
  agent_id: string;
  tags: string[];
  schema: Record<string, string>;
  classification: string;
  domain: string;
  status: string;
  steward: string;
  quality_score: number;
  freshness?: FreshnessInfo;
  popularity?: PopularityInfo;
  profile?: ProfileInfo;
  columns?: Column[];
  created_at: string;
  updated_at: string;
}

export interface LineageGraphNode {
  id: string;
  name: string;
  type: string;
  domain: string;
  classification: string;
  quality_score: number;
  status: string;
}

export interface LineageGraphEdge {
  id: string;
  source_id: string;
  target_id: string;
  transform_type: string;
  agent_id: string;
  description: string;
}

export interface LineageGraph {
  nodes: LineageGraphNode[];
  edges: LineageGraphEdge[];
}

export interface LineageNode {
  id: string;
  type: 'agent' | 'database' | 'storage' | 'external_api' | 'tool';
}

export interface LineageEdge {
  source: string;
  target: string;
  label: string;
  span_count: number;
}

export interface GlossaryTerm {
  id: string;
  tenant_id: string;
  term: string;
  definition: string;
  domain: string;
  owner: string;
  related_terms: string[];
  linked_assets: string[];
  created_at: string;
  updated_at: string;
}

export interface SearchResult {
  id: string;
  name: string;
  type: string;
  description: string;
  domain: string;
  classification: string;
  relevance: number;
  match_field: string;
}

export interface CatalogStats {
  total_sources: number;
  by_type: Record<string, number>;
  by_classification: Record<string, number>;
  by_domain: DomainStat[];
  avg_quality_score: number;
  stale_sources: number;
  pii_sources: number;
  total_columns: number;
  total_lineage_edges: number;
  total_glossary_terms: number;
}

export interface DomainStat {
  domain: string;
  count: number;
  avg_quality: number;
}

export interface ImpactAnalysis {
  source_id: string;
  source_name: string;
  downstream_count: number;
  paths: ImpactPath[];
}

export interface ImpactPath {
  nodes: ImpactNode[];
  total_depth: number;
}

export interface ImpactNode {
  id: string;
  name: string;
  type: string;
  depth: number;
}

export interface ToolUsage {
  tool: string;
  agent_id: string;
  call_count: number;
  avg_duration_ms: number;
}
