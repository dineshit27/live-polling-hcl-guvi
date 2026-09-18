export interface PollOption {
  id: string;
  text: string;
  vote_count: number;
}

export interface Poll {
  id: string;
  owner_id: string;
  title: string;
  description?: string;
  status: 'active' | 'closed';
  options: PollOption[];
  total_votes: number;
  created_at: string;
  updated_at: string;
}

export type RealtimeEventType =
  | 'VOTE_CAST'
  | 'POLL_STATUS'
  | 'POLL_SNAPSHOT'
  | 'POLL_CREATED'
  | 'POLL_DELETED'
  | 'POLL_UPDATED';

export interface RealtimeVotePayload {
  event: RealtimeEventType;
  poll_id: string;
  option_id?: string;
  option_counts: Record<string, number>;
  total_votes: number;
  status: 'active' | 'closed';
  timestamp: string;
}

export interface RealtimeWorkspaceEvent {
  event: RealtimeEventType;
  poll?: Poll;
  poll_id?: string;
  option_id?: string;
  option_counts?: Record<string, number>;
  total_votes?: number;
  status?: 'active' | 'closed';
  timestamp?: string;
}

export type RealtimeConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'disconnected';

export interface User {
  id: string;
  name: string;
  email: string;
  created_at?: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface ApiResponse<T = unknown> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
  details?: Record<string, string>;
}

export interface VoteStatus {
  has_voted: boolean;
  voted_option_id?: string;
}

export interface HealthCheckResponse {
  status: string;
  timestamp?: string;
  services?: {
    mongodb?: string;
    redis?: string;
  };
  message?: string;
}
