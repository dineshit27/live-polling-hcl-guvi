import { useEffect, useRef, useState, useCallback } from 'react';
import { Poll, RealtimeConnectionState, RealtimeVotePayload } from '../types';

interface UsePollRealtimeOptions {
  pollId: string | null;
  initialPoll?: Poll | null;
}

interface UsePollRealtimeReturn {
  poll: Poll | null;
  connectionState: RealtimeConnectionState;
  reconnectAttempts: number;
  lastUpdated: Date | null;
  error: string | null;
  refreshState: () => Promise<void>;
}

// Runtime validator for incoming realtime event payloads
function isValidPayload(data: unknown, currentPollId: string): data is RealtimeVotePayload {
  if (!data || typeof data !== 'object') return false;
  const p = data as Record<string, unknown>;

  // Validate event type
  if (typeof p.event !== 'string' || !['VOTE_CAST', 'POLL_STATUS', 'POLL_SNAPSHOT'].includes(p.event)) {
    return false;
  }

  // Validate poll ID matches current poll
  if (typeof p.poll_id !== 'string' || p.poll_id !== currentPollId) {
    return false;
  }

  // Validate option counts map
  if (!p.option_counts || typeof p.option_counts !== 'object' || Array.isArray(p.option_counts)) {
    return false;
  }

  // Validate total votes is a valid number
  if (typeof p.total_votes !== 'number' || isNaN(p.total_votes) || p.total_votes < 0) {
    return false;
  }

  // Validate status is active or closed
  if (p.status !== 'active' && p.status !== 'closed') {
    return false;
  }

  return true;
}

export function usePollRealtime({ pollId, initialPoll }: UsePollRealtimeOptions): UsePollRealtimeReturn {
  const [poll, setPoll] = useState<Poll | null>(initialPoll || null);
  const [connectionState, setConnectionState] = useState<RealtimeConnectionState>('connecting');
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [error, setError] = useState<string | null>(null);

  const eventSourceRef = useRef<EventSource | null>(null);
  const retryTimeoutRef = useRef<number | null>(null);
  const mountedRef = useRef(true);
  const attemptsRef = useRef(0);

  // Sync initialPoll if updated externally
  useEffect(() => {
    if (initialPoll) {
      setPoll(initialPoll);
    }
  }, [initialPoll]);

  // Full reconciliation with server database (called on load & after reconnection)
  const refreshState = useCallback(async () => {
    if (!pollId) return;
    try {
      const res = await fetch(`/api/polls/${pollId}`);
      if (!res.ok) throw new Error(`Failed to fetch poll (${res.status})`);
      const json = await res.json();
      if (mountedRef.current && json.data) {
        setPoll(json.data);
        setLastUpdated(new Date());
      }
    } catch (err) {
      console.warn('[Realtime] Failed to sync poll snapshot:', err);
    }
  }, [pollId]);

  // Helper to apply incoming validated realtime payloads directly into local poll state
  const applyRealtimePayload = useCallback((payload: RealtimeVotePayload) => {
    setPoll((prev) => {
      if (!prev || prev.id !== payload.poll_id) return prev;

      // Update options with newly received counts while preserving text and structure
      const updatedOptions = prev.options.map((opt) => {
        const count = payload.option_counts[opt.id];
        return {
          ...opt,
          vote_count: typeof count === 'number' ? count : opt.vote_count,
        };
      });

      return {
        ...prev,
        options: updatedOptions,
        total_votes: payload.total_votes,
        status: payload.status,
        updated_at: payload.timestamp || new Date().toISOString(),
      };
    });
    setLastUpdated(new Date());
  }, []);

  useEffect(() => {
    mountedRef.current = true;

    if (!pollId) {
      setConnectionState('disconnected');
      return;
    }

    // Clean up any existing connection before establishing a new one
    const cleanupConnection = () => {
      if (retryTimeoutRef.current) {
        window.clearTimeout(retryTimeoutRef.current);
        retryTimeoutRef.current = null;
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };

    cleanupConnection();

    function connect() {
      if (!mountedRef.current || !pollId) return;

      const url = `/api/polls/${pollId}/events`;
      setConnectionState((prev) => (attemptsRef.current > 0 ? 'reconnecting' : 'connecting'));
      setError(null);

      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.onopen = () => {
        if (!mountedRef.current) return;
        setConnectionState('connected');
        setError(null);
        // If reconnecting after a drop, reconcile state with server to catch any missed votes
        if (attemptsRef.current > 0) {
          refreshState();
        }
        attemptsRef.current = 0;
        setReconnectAttempts(0);
      };

      // Handler for VOTE_CAST events
      es.addEventListener('VOTE_CAST', (e: MessageEvent) => {
        if (!mountedRef.current) return;
        try {
          const parsed = JSON.parse(e.data);
          if (isValidPayload(parsed, pollId)) {
            applyRealtimePayload(parsed);
          } else {
            console.warn('[Realtime] Ignored malformed VOTE_CAST payload:', e.data);
          }
        } catch (err) {
          console.warn('[Realtime] Failed to parse VOTE_CAST JSON:', err);
        }
      });

      // Handler for POLL_STATUS events (e.g. poll closed)
      es.addEventListener('POLL_STATUS', (e: MessageEvent) => {
        if (!mountedRef.current) return;
        try {
          const parsed = JSON.parse(e.data);
          if (isValidPayload(parsed, pollId)) {
            applyRealtimePayload(parsed);
          } else {
            console.warn('[Realtime] Ignored malformed POLL_STATUS payload:', e.data);
          }
        } catch (err) {
          console.warn('[Realtime] Failed to parse POLL_STATUS JSON:', err);
        }
      });

      // Handler for POLL_SNAPSHOT events (initial state on connection)
      es.addEventListener('POLL_SNAPSHOT', (e: MessageEvent) => {
        if (!mountedRef.current) return;
        try {
          const parsed = JSON.parse(e.data);
          if (isValidPayload(parsed, pollId)) {
            applyRealtimePayload(parsed);
          }
        } catch (err) {
          console.warn('[Realtime] Failed to parse POLL_SNAPSHOT JSON:', err);
        }
      });

      es.onerror = () => {
        if (!mountedRef.current) return;
        es.close();
        eventSourceRef.current = null;

        // Max 5 retry attempts with exponential backoff (1s, 2s, 4s, 8s, max 10s)
        const maxAttempts = 5;
        attemptsRef.current += 1;
        setReconnectAttempts(attemptsRef.current);

        if (attemptsRef.current <= maxAttempts) {
          setConnectionState('reconnecting');
          const delay = Math.min(1000 * Math.pow(2, attemptsRef.current - 1), 10000);
          retryTimeoutRef.current = window.setTimeout(() => {
            if (mountedRef.current) {
              connect();
            }
          }, delay);
        } else {
          setConnectionState('disconnected');
          setError('Live connection lost. Click reconnect to resume real-time updates.');
        }
      };
    }

    // Initial fetch of poll data if not already present
    refreshState();
    // Establish real-time SSE connection
    connect();

    return () => {
      mountedRef.current = false;
      cleanupConnection();
    };
  }, [pollId, refreshState, applyRealtimePayload]);

  return {
    poll,
    connectionState,
    reconnectAttempts,
    lastUpdated,
    error,
    refreshState,
  };
}
