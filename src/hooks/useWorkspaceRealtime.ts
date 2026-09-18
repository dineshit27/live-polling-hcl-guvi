import { useEffect, useRef, useState, useCallback } from 'react';
import { Poll, RealtimeConnectionState, RealtimeWorkspaceEvent } from '../types';

interface UseWorkspaceRealtimeOptions {
  onPollCreated?: (poll: Poll) => void;
  onVoteCast?: (payload: {
    poll_id: string;
    option_id?: string;
    option_counts?: Record<string, number>;
    total_votes?: number;
    status?: 'active' | 'closed';
  }) => void;
  onPollStatus?: (payload: {
    poll_id: string;
    status: 'active' | 'closed';
    option_counts?: Record<string, number>;
    total_votes?: number;
  }) => void;
  onPollDeleted?: (pollId: string) => void;
  onPollUpdated?: (poll: Poll) => void;
  onRefreshNeeded?: () => void;
}

export function useWorkspaceRealtime(options: UseWorkspaceRealtimeOptions = {}) {
  const [connectionState, setConnectionState] = useState<RealtimeConnectionState>('connecting');
  const [lastEventTime, setLastEventTime] = useState<Date | null>(null);

  const optionsRef = useRef(options);
  optionsRef.current = options;

  const esRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);

  const connect = useCallback(() => {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }

    setConnectionState((prev) => (reconnectAttemptsRef.current > 0 ? 'reconnecting' : 'connecting'));
    const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'https://live-polling-hcl-guvi.onrender.com';
    const es = new EventSource(`${BASE_URL}/api/workspace/events`);
    esRef.current = es;

    es.onopen = () => {
      setConnectionState('connected');
      // If reconnected, tell the workspace to refresh state to catch any missed updates
      if (reconnectAttemptsRef.current > 0) {
        optionsRef.current.onRefreshNeeded?.();
      }
      reconnectAttemptsRef.current = 0;
    };

    es.onerror = () => {
      es.close();
      esRef.current = null;
      setConnectionState('reconnecting');

      const delay = Math.min(1000 * 2 ** reconnectAttemptsRef.current, 10000);
      reconnectAttemptsRef.current += 1;

      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = window.setTimeout(() => {
        connect();
      }, delay);
    };

    // 0. CONNECTED confirmation event from backend
    es.addEventListener('CONNECTED', () => {
      setConnectionState('connected');
      if (reconnectAttemptsRef.current > 0) {
        optionsRef.current.onRefreshNeeded?.();
      }
      reconnectAttemptsRef.current = 0;
    });

    // 1. POLL_CREATED
    es.addEventListener('POLL_CREATED', (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as RealtimeWorkspaceEvent;
        if (data.poll) {
          optionsRef.current.onPollCreated?.(data.poll);
          setLastEventTime(new Date());
        }
      } catch (err) {
        console.warn('[WorkspaceRealtime] Error parsing POLL_CREATED:', err);
      }
    });

    // 2. VOTE_CAST
    es.addEventListener('VOTE_CAST', (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as RealtimeWorkspaceEvent;
        if (data.poll_id) {
          optionsRef.current.onVoteCast?.({
            poll_id: data.poll_id,
            option_id: data.option_id,
            option_counts: data.option_counts,
            total_votes: data.total_votes,
            status: data.status,
          });
          setLastEventTime(new Date());
        }
      } catch (err) {
        console.warn('[WorkspaceRealtime] Error parsing VOTE_CAST:', err);
      }
    });

    // 3. POLL_STATUS (e.g. poll closed)
    es.addEventListener('POLL_STATUS', (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as RealtimeWorkspaceEvent;
        if (data.poll_id && data.status) {
          optionsRef.current.onPollStatus?.({
            poll_id: data.poll_id,
            status: data.status,
            option_counts: data.option_counts,
            total_votes: data.total_votes,
          });
          setLastEventTime(new Date());
        }
      } catch (err) {
        console.warn('[WorkspaceRealtime] Error parsing POLL_STATUS:', err);
      }
    });

    // 4. POLL_DELETED
    es.addEventListener('POLL_DELETED', (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as RealtimeWorkspaceEvent;
        if (data.poll_id) {
          optionsRef.current.onPollDeleted?.(data.poll_id);
          setLastEventTime(new Date());
        }
      } catch (err) {
        console.warn('[WorkspaceRealtime] Error parsing POLL_DELETED:', err);
      }
    });

    // 5. POLL_UPDATED
    es.addEventListener('POLL_UPDATED', (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as RealtimeWorkspaceEvent;
        if (data.poll) {
          optionsRef.current.onPollUpdated?.(data.poll);
          setLastEventTime(new Date());
        }
      } catch (err) {
        console.warn('[WorkspaceRealtime] Error parsing POLL_UPDATED:', err);
      }
    });
  }, []);

  useEffect(() => {
    connect();

    const handleOnlineOrFocus = () => {
      if (!esRef.current || esRef.current.readyState === EventSource.CLOSED) {
        reconnectAttemptsRef.current = 0;
        connect();
      }
    };

    window.addEventListener('focus', handleOnlineOrFocus);
    window.addEventListener('online', handleOnlineOrFocus);

    return () => {
      window.removeEventListener('focus', handleOnlineOrFocus);
      window.removeEventListener('online', handleOnlineOrFocus);
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (esRef.current) {
        esRef.current.close();
        esRef.current = null;
      }
    };
  }, [connect]);

  return {
    connectionState,
    isRealtime: connectionState === 'connected',
    lastEventTime,
  };
}
