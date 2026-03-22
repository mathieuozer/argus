# Dashboard Guide

The Argus dashboard is a React 18 + TypeScript application built with Vite. It provides a comprehensive UI for monitoring agents, tasks, telemetry, and platform health.

## Tech Stack

| Technology | Purpose |
|---|---|
| React 18 | UI framework |
| TypeScript | Type safety |
| Vite 6 | Build tool and dev server |
| Zustand | State management |
| Recharts | Charts and visualizations |
| React Router | Client-side routing |
| Tailwind CSS | Styling |

## Running

### Development (with Mock API)

```bash
cd dashboard
npm install
npm run dev
```

Opens at http://localhost:5173. Uses the built-in mock API plugin for standalone development.

### Development (with Real Backend)

```bash
ARGUS_API_URL=http://localhost:8080 npm run dev
```

This enables Vite's proxy to forward `/api/*` to the gateway and `/ws/*` for WebSocket streams.

### Production Build

```bash
npm run build
```

Output in `dashboard/dist/`. Serve with any static file server.

## Pages

### Core Operations

| Page | Route | Description |
|---|---|---|
| Login | `/login` | Authentication with JWT token |
| Agents | `/agents` | Agent inventory with status cards (healthy/degraded/failed/quarantined) |
| Agent Detail | `/agents/:id` | Individual agent metrics, tasks, telemetry spans |
| Tasks | `/tasks` | Task queue with filtering by status and agent |
| Metrics | `/metrics` | System-wide metrics: agent count, task throughput, cost, latency |
| Alerts | `/alerts` | Alert feed including predictive failure warnings |

### Observability

| Page | Route | Description |
|---|---|---|
| Traces | `/traces` | Distributed trace search and list |
| Trace Detail | `/traces/:id` | Waterfall view of spans within a trace |
| Data Quality | `/data-quality` | Quality scores, rules, and violations |
| Catalog | `/catalog` | Discovered data sources and lineage graph |
| Cost | `/costs` | Cost governance: breakdown, trends, budgets, anomalies |
| SLOs | `/slos` | SLO definitions, error budgets, compliance percentages |
| Audit Log | `/audit` | Immutable audit trail with search and filtering |

### AI Operations

| Page | Route | Description |
|---|---|---|
| Evaluations | `/evals` | Agent benchmark suites, runs, and results |
| Feedback | `/feedback` | Human feedback collection with ratings and tags |
| Guardrails | `/guardrails` | Safety rules, violations, and statistics |
| Prompts | `/prompts` | Prompt template versioning and metrics |
| RAG | `/rag` | Retrieval-augmented generation analytics |
| Compliance | `/compliance` | Compliance reports (GDPR, NIS2, KVKK, FedRAMP) |
| Playground | `/playground` | Interactive agent testing |

### Administration

| Page | Route | Description |
|---|---|---|
| Settings | `/settings` | Tenant configuration, policy editor, RBAC |

## Zustand Stores

All global state is managed via Zustand stores in `dashboard/src/stores/`.

| Store | File | Manages |
|---|---|---|
| `useAgentStore` | `agentStore.ts` | Agent list, registration, status updates |
| `useTaskStore` | `taskStore.ts` | Task list, creation, status tracking |
| `useAuthStore` | `authStore.ts` | JWT token, login/logout, session |
| `useAlertStore` | `alertStore.ts` | Alert feed, acknowledgment |
| `useMetricsStore` | `metricsStore.ts` | System metrics (agents, tasks, cost) |
| `useTraceStore` | `traceStore.ts` | Trace queries, span details |
| `useAuditStore` | `auditStore.ts` | Audit log entries, search |
| `useDataQualityStore` | `dataQualityStore.ts` | Quality rules, scores, violations |
| `useCatalogStore` | `catalogStore.ts` | Data source catalog, lineage |
| `useCostStore` | `costStore.ts` | Cost breakdown, trends, budgets |
| `useSloStore` | `sloStore.ts` | SLO definitions, status |
| `useEvalStore` | `evalStore.ts` | Evaluation suites, runs |
| `useFeedbackStore` | `feedbackStore.ts` | Feedback submissions, summaries |
| `useGuardrailStore` | `guardrailStore.ts` | Guardrail rules, violations |
| `usePromptStore` | `promptStore.ts` | Prompt templates, versions |
| `useRagStore` | `ragStore.ts` | RAG retrievals, sources |
| `useComplianceStore` | `complianceStore.ts` | Compliance reports |

## API Client

All API calls go through `dashboard/src/utils/apiClient.ts`. This singleton:
- Prepends `/api/v1` to all paths
- Adds `X-Tenant-ID` header (default: `"default"`)
- Adds `Authorization: Bearer <token>` when logged in
- Handles 401 by redirecting to login
- Returns typed `ApiResponse<T>` objects

Usage in stores:
```typescript
import apiClient from '../utils/apiClient';

const response = await apiClient.get<Agent[]>('/agents');
// response.data is Agent[]
```

## Components

Components are organized by domain in `dashboard/src/components/`:

```
components/
├── agents/     # AgentCard, AgentList, AgentStatusBadge
├── metrics/    # MetricCard, SparklineChart, CostPanel
├── alerts/     # AlertFeed, PredictiveWarningCard
├── settings/   # TenantConfig, PolicyEditor, RBACPanel
└── layout/     # Navbar, Sidebar, AppShell
```

## Types

All TypeScript interfaces live in `dashboard/src/types/`:

- `api.ts` - `ApiResponse<T>`, `ApiError`, pagination types
- `agent.ts` - `Agent`, `AgentStatus`, `AgentFramework`
- `task.ts` - `Task`, `TaskStatus`
- `telemetry.ts` - `TelemetrySpan`, `DataTier`
- `alert.ts` - `PredictiveAlert`, `AlertStatus`

## Internationalization (i18n)

The dashboard supports:
- English (default)
- Arabic (with RTL layout)

i18n is managed via React context. Language files are in `dashboard/src/i18n/`.

## Conventions

- **Components**: Functional only, one default export per file, filename matches component name
- **State**: Zustand for global, `useState` for component-local
- **API calls**: Always through `apiClient`, never raw `fetch`
- **Types**: In `src/types/`, never inline in components
- **Routing**: React Router v6 with lazy-loaded pages
