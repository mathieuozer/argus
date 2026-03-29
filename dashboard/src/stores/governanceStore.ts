import { create } from 'zustand';
import type { ClassificationPolicy, RetentionPolicy, AccessLog, PIIScanResult, ComplianceMapping, DataSteward, GovernanceSummary } from '../types/governance';
import apiClient from '../utils/apiClient';

interface GovernanceState {
  summary: GovernanceSummary | null;
  classificationPolicies: ClassificationPolicy[];
  retentionPolicies: RetentionPolicy[];
  accessLogs: AccessLog[];
  piiScans: PIIScanResult[];
  complianceMappings: ComplianceMapping[];
  stewards: DataSteward[];
  loading: boolean;
  error: string | null;
  fetchSummary: () => Promise<void>;
  fetchClassificationPolicies: () => Promise<void>;
  fetchRetentionPolicies: () => Promise<void>;
  fetchAccessLogs: (sourceId?: string, agentId?: string) => Promise<void>;
  fetchPIIScans: () => Promise<void>;
  fetchComplianceMappings: (framework?: string, status?: string) => Promise<void>;
  fetchStewards: () => Promise<void>;
  createClassificationPolicy: (policy: Partial<ClassificationPolicy>) => Promise<void>;
  deleteClassificationPolicy: (id: string) => Promise<void>;
  createRetentionPolicy: (policy: Partial<RetentionPolicy>) => Promise<void>;
  deleteRetentionPolicy: (id: string) => Promise<void>;
  createSteward: (steward: Partial<DataSteward>) => Promise<void>;
  deleteSteward: (id: string) => Promise<void>;
  updateComplianceMapping: (id: string, status: string, evidence: string[]) => Promise<void>;
}

export const useGovernanceStore = create<GovernanceState>((set, get) => ({
  summary: null,
  classificationPolicies: [],
  retentionPolicies: [],
  accessLogs: [],
  piiScans: [],
  complianceMappings: [],
  stewards: [],
  loading: false,
  error: null,
  fetchSummary: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<GovernanceSummary>('/governance/summary');
      set({ summary: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchClassificationPolicies: async () => {
    try {
      const response = await apiClient.get<ClassificationPolicy[]>('/governance/classification-policies');
      set({ classificationPolicies: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchRetentionPolicies: async () => {
    try {
      const response = await apiClient.get<RetentionPolicy[]>('/governance/retention-policies');
      set({ retentionPolicies: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchAccessLogs: async (sourceId?: string, agentId?: string) => {
    try {
      const query = new URLSearchParams();
      if (sourceId) query.set('source_id', sourceId);
      if (agentId) query.set('agent_id', agentId);
      const qs = query.toString();
      const path = qs ? `/governance/access-logs?${qs}` : '/governance/access-logs';
      const response = await apiClient.get<AccessLog[]>(path);
      set({ accessLogs: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchPIIScans: async () => {
    try {
      const response = await apiClient.get<PIIScanResult[]>('/governance/pii-scans');
      set({ piiScans: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchComplianceMappings: async (framework?: string, status?: string) => {
    try {
      const query = new URLSearchParams();
      if (framework) query.set('framework', framework);
      if (status) query.set('status', status);
      const qs = query.toString();
      const path = qs ? `/governance/compliance-mappings?${qs}` : '/governance/compliance-mappings';
      const response = await apiClient.get<ComplianceMapping[]>(path);
      set({ complianceMappings: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchStewards: async () => {
    try {
      const response = await apiClient.get<DataSteward[]>('/governance/stewards');
      set({ stewards: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  createClassificationPolicy: async (policy) => {
    try {
      await apiClient.post('/governance/classification-policies', policy);
      await get().fetchClassificationPolicies();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  deleteClassificationPolicy: async (id: string) => {
    try {
      await apiClient.del(`/governance/classification-policies/${id}`);
      await get().fetchClassificationPolicies();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  createRetentionPolicy: async (policy) => {
    try {
      await apiClient.post('/governance/retention-policies', policy);
      await get().fetchRetentionPolicies();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  deleteRetentionPolicy: async (id: string) => {
    try {
      await apiClient.del(`/governance/retention-policies/${id}`);
      await get().fetchRetentionPolicies();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  createSteward: async (steward) => {
    try {
      await apiClient.post('/governance/stewards', steward);
      await get().fetchStewards();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  deleteSteward: async (id: string) => {
    try {
      await apiClient.del(`/governance/stewards/${id}`);
      await get().fetchStewards();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  updateComplianceMapping: async (id: string, status: string, evidence: string[]) => {
    try {
      await apiClient.put(`/governance/compliance-mappings/${id}`, { status, evidence });
      await get().fetchComplianceMappings();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
