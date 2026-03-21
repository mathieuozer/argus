export interface Retrieval {
  id: string;
  tenant_id: string;
  agent_id: string;
  span_id: string;
  query: string;
  num_chunks: number;
  avg_relevance: number;
  latency_ms: number;
  source_ids: string[];
  created_at: string;
}

export interface RAGSource {
  id: string;
  tenant_id: string;
  name: string;
  type: 'document' | 'database' | 'api';
  total_chunks: number;
  avg_relevance: number;
  usage_count: number;
}

export interface QualityTrend {
  timestamp: string;
  avg_relevance: number;
  avg_latency_ms: number;
  total_queries: number;
}
