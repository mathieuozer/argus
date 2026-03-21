export interface Feedback {
  id: string;
  tenant_id: string;
  agent_id: string;
  span_id: string;
  task_id: string;
  rating: number;
  comment: string;
  user_id: string;
  created_at: string;
}

export interface FeedbackSummary {
  agent_id: string;
  total_feedback: number;
  average_rating: number;
  positive_count: number;
  negative_count: number;
}
