import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Poll, User } from '../types';
import { api, ApiError } from '../services/api';
import { useToast } from './Toast';
import { PollCard } from './PollCard';
import { StatsBar } from './StatsBar';
import { useWorkspaceRealtime } from '../hooks/useWorkspaceRealtime';
import {
  Plus,
  Search,
  SlidersHorizontal,
  RefreshCw,
  Sparkles,
  Radio,
  X,
  Inbox,
  Filter,
} from 'lucide-react';

export type PollFilterType = 'all' | 'active' | 'closed' | 'my';
export type PollSortType = 'newest' | 'votes';

interface PollListProps {
  onSelectPoll: (pollId: string) => void;
  onCreatePollClick: () => void;
  currentUser?: User | null;
  onOpenDeleteModal?: (poll: Poll) => void;
  onPollsLoaded?: (polls: Poll[]) => void;
}

export const PollList: React.FC<PollListProps> = ({
  onSelectPoll,
  onCreatePollClick,
  currentUser,
  onOpenDeleteModal,
  onPollsLoaded,
}) => {
  const { showToast } = useToast();
  const [allPolls, setAllPolls] = useState<Poll[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters and search state
  const [filter, setFilter] = useState<PollFilterType>('all');
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState<PollSortType>('newest');

  const onPollsLoadedRef = useRef(onPollsLoaded);
  onPollsLoadedRef.current = onPollsLoaded;

  const currentUserId = currentUser?.id;

  const fetchPolls = useCallback(async (silent = false) => {
    try {
      if (!silent) setLoading(true);
      setError(null);

      let data: Poll[];
      if (filter === 'my') {
        if (!currentUserId) {
          setAllPolls([]);
          if (!silent) setLoading(false);
          return;
        }
        data = await api.getMyPolls();
      } else {
        data = await api.getActivePolls(100);
      }

      setAllPolls(data);
      onPollsLoadedRef.current?.(data);
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : 'Failed to load polls';
      setError(msg);
      showToast(msg, 'error', 'Error Loading Polls');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [filter, currentUserId, showToast]);

  useEffect(() => {
    fetchPolls();
  }, [fetchPolls]);

  // Real-time workspace live event callbacks
  const handlePollCreated = useCallback((newPoll: Poll) => {
    setAllPolls((prev) => {
      if (prev.some((p) => p.id === newPoll.id)) return prev;
      const updated = [newPoll, ...prev];
      onPollsLoadedRef.current?.(updated);
      return updated;
    });
  }, []);

  const handleVoteCast = useCallback((payload: {
    poll_id: string;
    option_id?: string;
    option_counts?: Record<string, number>;
    total_votes?: number;
    status?: 'active' | 'closed';
  }) => {
    setAllPolls((prev) => {
      const updated = prev.map((poll) => {
        if (poll.id !== payload.poll_id) return poll;
        const newOptions = poll.options.map((opt) => ({
          ...opt,
          vote_count: payload.option_counts?.[opt.id] ?? opt.vote_count,
        }));
        return {
          ...poll,
          options: newOptions,
          total_votes: typeof payload.total_votes === 'number' ? payload.total_votes : poll.total_votes,
          status: payload.status ?? poll.status,
          updated_at: new Date().toISOString(),
        };
      });
      onPollsLoadedRef.current?.(updated);
      return updated;
    });
  }, []);

  const handlePollStatus = useCallback((payload: {
    poll_id: string;
    status: 'active' | 'closed';
    option_counts?: Record<string, number>;
    total_votes?: number;
  }) => {
    setAllPolls((prev) => {
      const updated = prev.map((poll) => {
        if (poll.id !== payload.poll_id) return poll;
        return {
          ...poll,
          status: payload.status,
          total_votes: typeof payload.total_votes === 'number' ? payload.total_votes : poll.total_votes,
          updated_at: new Date().toISOString(),
        };
      });
      onPollsLoadedRef.current?.(updated);
      return updated;
    });
  }, []);

  const handlePollDeleted = useCallback((pollId: string) => {
    setAllPolls((prev) => {
      const updated = prev.filter((p) => p.id !== pollId);
      onPollsLoadedRef.current?.(updated);
      return updated;
    });
  }, []);

  const handlePollUpdated = useCallback((updatedPoll: Poll) => {
    setAllPolls((prev) => {
      const updated = prev.map((p) => (p.id === updatedPoll.id ? updatedPoll : p));
      onPollsLoadedRef.current?.(updated);
      return updated;
    });
  }, []);

  const handleRefreshNeeded = useCallback(() => {
    fetchPolls(true);
  }, [fetchPolls]);

  // Connect to persistent SSE live stream for real-time workspace updates
  const { isRealtime } = useWorkspaceRealtime({
    onPollCreated: handlePollCreated,
    onVoteCast: handleVoteCast,
    onPollStatus: handlePollStatus,
    onPollDeleted: handlePollDeleted,
    onPollUpdated: handlePollUpdated,
    onRefreshNeeded: handleRefreshNeeded,
  });

  // Re-fetch when browser tab regains focus (user switches back)
  useEffect(() => {
    const handleFocus = () => fetchPolls(true);
    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, [fetchPolls]);

  // Client-side filtering and sorting
  const filteredAndSortedPolls = useMemo(() => {
    let result = [...allPolls];

    // Status filter
    if (filter === 'active') {
      result = result.filter((p) => p.status === 'active');
    } else if (filter === 'closed') {
      result = result.filter((p) => p.status === 'closed');
    }

    // Search query filter
    const query = search.trim().toLowerCase();
    if (query) {
      result = result.filter(
        (p) =>
          p.title.toLowerCase().includes(query) ||
          (p.description && p.description.toLowerCase().includes(query))
      );
    }

    // Sort order
    result.sort((a, b) => {
      if (sortBy === 'votes') {
        if (b.total_votes !== a.total_votes) {
          return b.total_votes - a.total_votes;
        }
      }
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });

    return result;
  }, [allPolls, filter, search, sortBy]);

  const handleCopyLink = (id: string) => {
    const url = `${window.location.origin}/#${id}`;
    navigator.clipboard.writeText(url).then(() => {
      showToast('Poll link copied to clipboard!', 'info', 'Link Copied');
    });
  };

  // Quick count for active & closed
  const counts = useMemo(() => {
    const active = allPolls.filter((p) => p.status === 'active').length;
    const closed = allPolls.filter((p) => p.status === 'closed').length;
    return { all: allPolls.length, active, closed };
  }, [allPolls]);

  return (
    <div id="poll-list-view" className="space-y-6">
      
      {/* Workspace Summary Bar */}
      <StatsBar polls={allPolls} isHealthy={!error} isRealtime={isRealtime} />

      {/* Control Strip: Search, Tabs & Sorting */}
      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 shadow-xl space-y-4 backdrop-blur-md">
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
          
          {/* Search Input */}
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
            <input
              id="poll-search-input"
              type="text"
              placeholder="Search polls by title or topic..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-10 pr-9 py-2 text-xs bg-slate-950 border border-slate-800 rounded-xl text-white placeholder:text-slate-500 focus:outline-none focus:border-sky-500 transition-colors"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch('')}
                aria-label="Clear search"
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-white cursor-pointer"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>

          {/* Sort Selector & Refresh */}
          <div className="flex items-center gap-2 self-end sm:self-auto">
            <div className="flex items-center gap-1.5 bg-slate-950 px-3 py-1.5 rounded-xl border border-slate-800">
              <SlidersHorizontal className="w-3.5 h-3.5 text-slate-500" />
              <span className="text-[11px] text-slate-400">Sort:</span>
              <select
                id="poll-sort-select"
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as PollSortType)}
                className="bg-transparent text-xs text-white focus:outline-none cursor-pointer font-medium"
              >
                <option value="newest" className="bg-slate-900 text-white">Newest First</option>
                <option value="votes" className="bg-slate-900 text-white">Most Voted</option>
              </select>
            </div>

            <button
              type="button"
              onClick={fetchPolls}
              title="Refresh polls"
              className="p-2 rounded-xl bg-slate-950 hover:bg-slate-800 text-slate-400 hover:text-white border border-slate-800 transition-colors cursor-pointer"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-sky-400' : ''}`} />
            </button>

            <button
              id="create-poll-btn"
              type="button"
              onClick={onCreatePollClick}
              className="px-4 py-2 rounded-xl bg-sky-500 hover:bg-sky-400 text-slate-950 text-xs font-semibold shadow-md transition-all flex items-center gap-1.5 cursor-pointer whitespace-nowrap"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Create Poll</span>
            </button>
          </div>
        </div>

        {/* Tab Filters */}
        <div className="flex items-center gap-2 pt-2 border-t border-slate-800/80 overflow-x-auto">
          <button
            type="button"
            id="tab-filter-all"
            onClick={() => setFilter('all')}
            className={`px-3.5 py-1.5 text-xs font-semibold rounded-xl transition-all cursor-pointer flex items-center gap-1.5 ${
              filter === 'all'
                ? 'bg-sky-500 text-slate-950 shadow-xs'
                : 'text-slate-400 hover:text-white hover:bg-slate-800'
            }`}
          >
            <span>All Polls</span>
            <span className={`text-[10px] px-1.5 py-0.2 rounded-full font-mono ${filter === 'all' ? 'bg-slate-950/20 text-slate-950' : 'bg-slate-800 text-slate-400'}`}>
              {counts.all}
            </span>
          </button>

          <button
            type="button"
            id="tab-filter-active"
            onClick={() => setFilter('active')}
            className={`px-3.5 py-1.5 text-xs font-semibold rounded-xl transition-all cursor-pointer flex items-center gap-1.5 ${
              filter === 'active'
                ? 'bg-emerald-500 text-slate-950 shadow-xs'
                : 'text-slate-400 hover:text-white hover:bg-slate-800'
            }`}
          >
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
            <span>Active</span>
            <span className={`text-[10px] px-1.5 py-0.2 rounded-full font-mono ${filter === 'active' ? 'bg-slate-950/20 text-slate-950' : 'bg-slate-800 text-slate-400'}`}>
              {counts.active}
            </span>
          </button>

          <button
            type="button"
            id="tab-filter-closed"
            onClick={() => setFilter('closed')}
            className={`px-3.5 py-1.5 text-xs font-semibold rounded-xl transition-all cursor-pointer flex items-center gap-1.5 ${
              filter === 'closed'
                ? 'bg-slate-700 text-white shadow-xs'
                : 'text-slate-400 hover:text-white hover:bg-slate-800'
            }`}
          >
            <span>Closed</span>
            <span className={`text-[10px] px-1.5 py-0.2 rounded-full font-mono ${filter === 'closed' ? 'bg-slate-900 text-slate-200' : 'bg-slate-800 text-slate-400'}`}>
              {counts.closed}
            </span>
          </button>

          {currentUser && (
            <button
              type="button"
              id="tab-filter-my"
              onClick={() => setFilter('my')}
              className={`px-3.5 py-1.5 text-xs font-semibold rounded-xl transition-all cursor-pointer flex items-center gap-1.5 ml-auto ${
                filter === 'my'
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                  : 'text-slate-400 hover:text-white hover:bg-slate-800'
              }`}
            >
              <Sparkles className="w-3.5 h-3.5 text-sky-400" />
              <span>My Polls</span>
            </button>
          )}
        </div>
      </div>

      {/* Poll Cards Grid or States */}
      {loading ? (
        <div className="p-16 text-center bg-slate-900/60 border border-slate-800 rounded-2xl space-y-3">
          <div className="inline-block animate-spin w-8 h-8 border-3 border-sky-500 border-t-transparent rounded-full"></div>
          <p className="text-slate-400 text-xs font-medium">Loading live polling workspace...</p>
        </div>
      ) : error ? (
        <div className="p-8 bg-slate-900/80 border border-rose-800/60 rounded-2xl text-center space-y-4">
          <p className="text-rose-400 text-sm font-medium">{error}</p>
          <button
            type="button"
            onClick={fetchPolls}
            className="px-4 py-2 text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-white rounded-xl transition-colors cursor-pointer"
          >
            Retry Connection
          </button>
        </div>
      ) : filteredAndSortedPolls.length === 0 ? (
        <div className="p-16 text-center bg-slate-900/60 border border-slate-800 rounded-2xl space-y-4">
          <div className="w-12 h-12 rounded-2xl bg-slate-800/80 border border-slate-700/80 flex items-center justify-center text-slate-400 mx-auto">
            <Inbox className="w-6 h-6" />
          </div>
          <div>
            <h3 className="text-base font-bold text-white">
              {filter === 'my'
                ? 'No polls created yet'
                : search
                ? `No polls matching "${search}"`
                : filter === 'active'
                ? 'No active polls at this moment'
                : 'No polls found'}
            </h3>
            <p className="text-xs text-slate-400 max-w-sm mx-auto mt-1 leading-relaxed">
              {filter === 'my'
                ? 'Create your first live poll and watch votes arrive with zero refresh.'
                : search
                ? 'Try adjusting your search query or clear the filter.'
                : 'Publish a new question to start streaming realtime responses.'}
            </p>
          </div>

          <div className="pt-2">
            {filter === 'my' || !search ? (
              <button
                type="button"
                onClick={onCreatePollClick}
                className="px-5 py-2.5 text-xs font-semibold rounded-xl bg-sky-500 hover:bg-sky-400 text-slate-950 transition-colors cursor-pointer"
              >
                Create your first poll
              </button>
            ) : (
              <button
                type="button"
                onClick={() => setSearch('')}
                className="px-4 py-2 text-xs font-medium rounded-xl bg-slate-800 hover:bg-slate-700 text-white transition-colors cursor-pointer"
              >
                Clear Search
              </button>
            )}
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {filteredAndSortedPolls.map((poll) => {
            const isOwner = currentUser && currentUser.id === poll.owner_id;
            return (
              <PollCard
                key={poll.id}
                poll={poll}
                isOwner={!!isOwner}
                onSelect={onSelectPoll}
                onDeleteClick={onOpenDeleteModal}
                onCopyLink={handleCopyLink}
              />
            );
          })}
        </div>
      )}
    </div>
  );
};
