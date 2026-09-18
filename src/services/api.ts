import { ApiResponse, AuthResponse, Poll, User, VoteStatus } from '../types';

export class ApiError extends Error {
  code: string;
  status: number;
  details?: Record<string, string>;

  constructor(message: string, status = 500, code = 'unknown_error', details?: Record<string, string>) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

const AUTH_TOKEN_KEY = 'live_poll_token';

export function getStoredToken(): string | null {
  try {
    return localStorage.getItem(AUTH_TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setStoredToken(token: string | null): void {
  try {
    if (token) {
      localStorage.setItem(AUTH_TOKEN_KEY, token);
    } else {
      localStorage.removeItem(AUTH_TOKEN_KEY);
    }
  } catch {
    // ignore
  }
}
export const BACKEND_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'https://live-polling-hcl-guvi.onrender.com';
const BASE_URL = BACKEND_BASE_URL;

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
	const token = getStoredToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options.headers as Record<string, string>),
	};

	if (token && !headers['Authorization']) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	let res: Response;
	try {
		const fullUrl = endpoint.startsWith('http') ? endpoint : `${BASE_URL}${endpoint}`;
		res = await fetch(fullUrl, {
			...options,
			headers,
		});
	} catch {
		throw new ApiError('Unable to connect to server. Please check your connection.', 0, 'network_error');
	}

  let json: ApiResponse<T> | null = null;
  try {
    json = await res.json();
  } catch {
    // non-JSON response
  }

  if (!res.ok) {
    const errorMsg =
      (typeof json?.message === 'string' && json.message) ||
      (typeof json?.error === 'string' && json.error) ||
      `HTTP ${res.status} error`;
    const errorCode =
      (typeof json?.error === 'string' && json.error) ||
      `http_${res.status}`;
    throw new ApiError(errorMsg, res.status, errorCode, json?.details);
  }

  // Handle envelope unwrapping
  if (json && typeof json === 'object' && 'data' in json) {
    return (json.data !== undefined ? json.data : (json as unknown as T)) as T;
  }

  return (json !== null ? json : ({} as unknown as T)) as T;
}

export const api = {
  // Auth endpoints
  async signup(data: { name: string; email: string; password: string }): Promise<AuthResponse> {
    const res = await request<AuthResponse>('/api/auth/signup', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    if (res?.token) {
      setStoredToken(res.token);
    }
    return res;
  },

  async login(data: { email: string; password: string }): Promise<AuthResponse> {
    const res = await request<AuthResponse>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    if (res?.token) {
      setStoredToken(res.token);
    }
    return res;
  },

  async getCurrentUser(): Promise<User> {
    return request<User>('/api/auth/me', {
      method: 'GET',
    });
  },

  // Poll browsing and fetching
  async getActivePolls(limit = 50, offset = 0): Promise<Poll[]> {
    const polls = await request<Poll[]>(`/api/polls?limit=${limit}&offset=${offset}`, {
      method: 'GET',
    });
    return Array.isArray(polls) ? polls : [];
  },

  async getMyPolls(): Promise<Poll[]> {
    const polls = await request<Poll[]>('/api/polls/my', {
      method: 'GET',
    });
    return Array.isArray(polls) ? polls : [];
  },

  async getPoll(id: string): Promise<Poll> {
    return request<Poll>(`/api/polls/${id}`, {
      method: 'GET',
    });
  },

  // Poll management
  async createPoll(data: { title: string; description?: string; options: string[] }): Promise<Poll> {
    return request<Poll>('/api/polls', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async updatePoll(id: string, data: { title?: string; description?: string }): Promise<Poll> {
    return request<Poll>(`/api/polls/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  async closePoll(id: string): Promise<Poll> {
    return request<Poll>(`/api/polls/${id}/close`, {
      method: 'POST',
    });
  },

  async deletePoll(id: string): Promise<{ success: boolean; message: string }> {
    return request<{ success: boolean; message: string }>(`/api/polls/${id}`, {
      method: 'DELETE',
    });
  },

  // Voting
  async getVoteStatus(pollId: string, voterIdentifier?: string): Promise<VoteStatus> {
    const query = voterIdentifier ? `?voter_identifier=${encodeURIComponent(voterIdentifier)}` : '';
    const res = await request<VoteStatus>(`/api/polls/${pollId}/vote-status${query}`, {
      method: 'GET',
    });
    return res || { has_voted: false };
  },

  async castVote(pollId: string, data: { option_id: string; voter_identifier?: string }): Promise<Poll> {
    return request<Poll>(`/api/polls/${pollId}/vote`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },
};
