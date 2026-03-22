export interface TestCase {
  id: string;
  name: string;
  input: string;
  expected_output: string;
  criteria: Record<string, string>;
  max_latency_ms: number;
}

export interface TestSuite {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  agent_id: string;
  test_cases: TestCase[];
  created_at: string;
  updated_at: string;
}

export interface CaseResult {
  test_case_id: string;
  test_case_name: string;
  status: 'passed' | 'failed' | 'error';
  actual_output: string;
  latency_ms: number;
  score: number;
  reason: string;
}

export interface EvalRun {
  id: string;
  tenant_id: string;
  suite_id: string;
  suite_name: string;
  agent_id: string;
  status: 'running' | 'completed' | 'failed';
  score: number;
  total_cases: number;
  passed_cases: number;
  failed_cases: number;
  results: CaseResult[];
  started_at: string;
  completed_at: string | null;
}
