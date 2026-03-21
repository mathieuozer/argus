export interface TestCase {
  id: string;
  name: string;
  input: string;
  expectedOutput: string;
  criteria: Record<string, string>;
  maxLatencyMs: number;
}

export interface TestSuite {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  agentId: string;
  testCases: TestCase[];
  createdAt: string;
  updatedAt: string;
}

export interface CaseResult {
  testCaseId: string;
  testCaseName: string;
  status: 'passed' | 'failed' | 'error';
  actualOutput: string;
  latencyMs: number;
  score: number;
  reason: string;
}

export interface EvalRun {
  id: string;
  tenantId: string;
  suiteId: string;
  suiteName: string;
  agentId: string;
  status: 'running' | 'completed' | 'failed';
  score: number;
  totalCases: number;
  passedCases: number;
  failedCases: number;
  results: CaseResult[];
  startedAt: string;
  completedAt: string | null;
}
