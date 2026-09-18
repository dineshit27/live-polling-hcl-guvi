import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Poll, User, VoteStatus } from '../types';
import { usePollRealtime } from '../hooks/usePollRealtime';
import { LiveIndicator } from './LiveIndicator';
import { getVoterIdentifier } from '../lib/voter';
import { api, ApiError } from '../services/api';
import { useToast } from './Toast';
import { DeletePollModal } from './DeletePollModal';
import { ClosePollModal } from './ClosePollModal';
import {
  CheckCircle2,
  Lock,
  ArrowLeft,
  Users,
  Calendar,
  Trash2,
  AlertCircle,
  RefreshCw,
  Loader2,
  Share2,
  Check,
  Award,
  BarChart2,
  Sparkles,
  Radio,
} from 'lucide-react';
import { useSEO, sanitizeMetaText } from '../hooks/useSEO';

interface LivePollViewProps {
  pollId: string;
  currentUser: User | null;
  authToken: string | null;
  onBack: () => void;
  onRequireAuth?: () => void;
  onPollDeleted?: (pollId: string) => void;
}

export const LivePollView: React.FC<LivePollViewProps> = ({
  pollId,
  currentUser,
  authToken,
  onBack,
  onRequireAuth,
  onPollDeleted,
}) => {
  const { showToast } = useToast();
  const {
    poll,
    connectionState,
    reconnectAttempts,
    lastUpdated,
    error: pollError,
    refreshState,
  } = usePollRealtime({ pollId });

  const [voteStatus, setVoteStatus] = useState<VoteStatus | null>(null);
  const [selectedOptionId, setSelectedOptionId] = useState<string | null>(null);
  const [isSubmittingVote, setIsSubmittingVote] = useState(false);
  const [voteError, setVoteError] = useState<string | null>(null);
  const [isClosingPoll, setIsClosingPoll] = useState(false);
  const [isCloseModalOpen, setIsCloseModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [copiedLink, setCopiedLink] = useState(false);
  const [recentlyUpdatedOptionId, setRecentlyUpdatedOptionId] = useState<string | null>(null);
  const [viewResultsMode, setViewResultsMode] = useState(false);

  // Dynamic SEO metadata for public poll
  const pollTitle = poll?.title ? sanitizeMetaText(poll.title) : '';
  const pageTitle = pollTitle
    ? `Vote: ${pollTitle} | PULSE Live`
    : 'Live Poll Stream | PULSE Live';
  const pageDescription = pollTitle
    ? `Vote in "${pollTitle}" and see results update in real time with PULSE Live.`
    : 'Vote in this live poll and see results update in real time with PULSE Live.';
  const pollCanonical = `https://pulse-live-hclguvi.onrender.com/#${pollId}`;

  useSEO({
    title: pageTitle,
    description: pageDescription,
    canonical: pollCanonical,
    robots: 'index, follow',
    enabled: true,
  });

  const prevCountsRef = useRef<Record<string, number>>({});
  const voterIdentifier = getVoterIdentifier();

  // Check if voter has already cast a vote
  const checkVoteStatus = useCallback(async () => {
    try {
      const status = await api.getVoteStatus(pollId, voterIdentifier);
      setVoteStatus(status);
      if (status.has_voted) {
        setViewResultsMode(true);
      }
    } catch {
      // Background check
    }
  }, [pollId, voterIdentifier]);

  useEffect(() => {
    checkVoteStatus();
  }, [checkVoteStatus]);

  // Detect which option changed dynamically when new votes stream in via SSE
  useEffect(() => {
    if (!poll) return;

    let changedId: string | null = null;
    if (Array.isArray(poll.options)) {
      poll.options.forEach((opt) => {
        const prevCount = prevCountsRef.current[opt.id] ?? opt.vote_count;
        if (opt.vote_count > prevCount) {
          changedId = opt.id;
        }
        prevCountsRef.current[opt.id] = opt.vote_count;
      });
    }

    if (changedId) {
      setRecentlyUpdatedOptionId(changedId);
      const timer = setTimeout(() => {
        setRecentlyUpdatedOptionId(null);
      }, 1500);
      return () => clearTimeout(timer);
    }
  }, [poll]);

  // Handle vote submission
  const handleCastVote = async () => {
    if (!poll || poll.status === 'closed' || !selectedOptionId || voteStatus?.has_voted) return;

    setIsSubmittingVote(true);
    setVoteError(null);

    try {
      await api.castVote(pollId, {
        option_id: selectedOptionId,
        voter_identifier: voterIdentifier,
      });

      setVoteStatus({ has_voted: true, voted_option_id: selectedOptionId });
      setViewResultsMode(true);
      showToast('Your vote was recorded and published to all clients via Redis Pub/Sub!', 'success', 'Vote Broadcast');
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 409 || err.code === 'already_voted') {
          setVoteStatus({ has_voted: true, voted_option_id: selectedOptionId });
          setViewResultsMode(true);
          setVoteError('You have already voted on this poll.');
          showToast('You have already voted on this poll.', 'warning', 'Duplicate Vote');
        } else if (err.status === 400 && err.code === 'poll_closed') {
          setVoteError('This poll has been closed.');
          showToast('This poll is closed.', 'warning', 'Poll Closed');
        } else {
          setVoteError(err.message);
          showToast(err.message, 'error', 'Vote Failed');
        }
      } else {
        const msg = 'Network error while casting vote';
        setVoteError(msg);
        showToast(msg, 'error', 'Connection Error');
      }
    } finally {
      setIsSubmittingVote(false);
    }
  };

  // Close poll (owner only)
  const handleClosePoll = async () => {
    if (!authToken || !poll) {
      onRequireAuth?.();
      return;
    }

    setIsClosingPoll(true);
    try {
      await api.closePoll(pollId);
      showToast('Poll closed. Voting has concluded.', 'info', 'Poll Closed');
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : 'Network error while closing poll';
      showToast(msg, 'error', 'Close Failed');
    } finally {
      setIsClosingPoll(false);
    }
  };

  const handleShare = () => {
    const url = window.location.href;
    navigator.clipboard.writeText(url).then(() => {
      setCopiedLink(true);
      showToast('Public poll URL copied to clipboard!', 'info', 'Link Copied');
      setTimeout(() => setCopiedLink(false), 2000);
    });
  };

  if (pollError && !poll) {
    return (
      <div className="max-w-2xl w-full mx-auto p-12 text-center bg-slate-900/90 border border-rose-800/60 rounded-2xl space-y-4">
        <div className="w-12 h-12 rounded-2xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400 mx-auto">
          <AlertCircle className="w-6 h-6" />
        </div>
        <h3 className="text-base font-semibold text-white">Unable to Load Poll</h3>
        <p className="text-rose-300 text-xs font-mono">{pollError}</p>
        <div className="flex items-center justify-center gap-3 pt-2">
          <button
            type="button"
            onClick={() => void refreshState()}
            className="px-4 py-2 text-xs font-semibold bg-sky-500 hover:bg-sky-400 text-slate-950 rounded-xl transition-colors cursor-pointer inline-flex items-center gap-1.5"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            <span>Retry</span>
          </button>
          <button
            type="button"
            onClick={onBack}
            className="px-4 py-2 text-xs text-slate-400 hover:text-white bg-slate-800 rounded-xl transition-colors cursor-pointer inline-flex items-center gap-1.5"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            <span>Return to All Polls</span>
          </button>
        </div>
      </div>
    );
  }

  if (!poll) {
    return (
      <div className="max-w-2xl w-full mx-auto p-12 text-center bg-slate-900/90 border border-slate-800 rounded-2xl space-y-4">
        <div className="inline-block animate-spin w-8 h-8 border-3 border-sky-500 border-t-transparent rounded-full mb-1"></div>
        <h3 className="text-base font-semibold text-white">Connecting to live poll stream...</h3>
        <p className="text-slate-400 text-xs font-mono">Subscribing to Redis channel poll:{pollId}:events via SSE</p>
        <div className="pt-2">
          <button
            type="button"
            onClick={onBack}
            className="px-4 py-2 text-xs text-slate-400 hover:text-white bg-slate-800 rounded-xl transition-colors cursor-pointer inline-flex items-center gap-1.5"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            <span>Return to All Polls</span>
          </button>
        </div>
      </div>
    );
  }

  const isOwner = currentUser && currentUser.id === poll.owner_id;
  const isClosed = poll.status === 'closed';
  const hasVoted = voteStatus?.has_voted;
  const votedOptionId = voteStatus?.voted_option_id;

  // Guard: ensure options is always a valid array before any iteration
  const safeOptions = Array.isArray(poll.options) ? poll.options : [];

  // Compute ranking
  const sortedOptions = [...safeOptions].sort((a, b) => b.vote_count - a.vote_count);
  const highestVoteCount = sortedOptions[0]?.vote_count || 0;

  const showResults = isClosed || hasVoted || viewResultsMode;

  return (
    <article id="live-poll-view" aria-labelledby="poll-title" className="max-w-2xl w-full mx-auto space-y-6">
      
      {/* Top Controls Bar */}
      <div className="flex items-center justify-between gap-4">
        <button
          id="back-to-polls-btn"
          type="button"
          onClick={onBack}
          aria-label="Return to all polls"
          className="inline-flex items-center gap-2 text-xs font-semibold text-slate-400 hover:text-white transition-colors cursor-pointer group"
        >
          <ArrowLeft className="w-4 h-4 transition-transform group-hover:-translate-x-1" />
          <span>All Polls</span>
        </button>

        <div className="flex items-center gap-2.5">
          <LiveIndicator
            status={connectionState}
            reconnectAttempts={reconnectAttempts}
            onReconnect={refreshState}
          />

          <span
            className={`px-2.5 py-0.5 text-xs font-bold rounded-full uppercase tracking-wider ${
              isClosed
                ? 'bg-slate-800 text-slate-400 border border-slate-700'
                : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/25'
            }`}
          >
            {isClosed ? 'Closed' : 'Active'}
          </span>

          <button
            type="button"
            onClick={handleShare}
            title="Share Poll"
            aria-label="Share public poll URL"
            className="p-1.5 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-300 hover:text-white transition-colors cursor-pointer"
          >
            {copiedLink ? <Check className="w-4 h-4 text-emerald-400" /> : <Share2 className="w-4 h-4" />}
          </button>
        </div>
      </div>

      {/* Main Poll Card (Centerpiece) */}
      <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-6 sm:p-8 shadow-2xl space-y-6 relative overflow-hidden backdrop-blur-md">
        
        {/* Card Header & Question */}
        <div>
          <div className="flex items-start justify-between gap-4">
            <h1 id="poll-title" className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight leading-snug">
              {poll.title}
            </h1>
            {isOwner && (
              <span className="shrink-0 px-2.5 py-1 text-[11px] font-semibold rounded-full bg-sky-500/10 text-sky-400 border border-sky-500/20">
                Created by you
              </span>
            )}
          </div>

          {poll.description && (
            <p className="text-slate-300 text-sm mt-2 leading-relaxed">
              {poll.description}
            </p>
          )}

          {/* Real-time Metadata Bar */}
          <div className="flex flex-wrap items-center gap-4 text-xs text-slate-400 mt-4 pt-4 border-t border-slate-800/80">
            <span className="flex items-center gap-1.5 font-mono">
              <Users className="w-3.5 h-3.5 text-sky-400" />
              <strong className="text-white text-sm font-bold">{poll.total_votes}</strong>{' '}
              {poll.total_votes === 1 ? 'vote' : 'votes'} total
            </span>

            <span className="flex items-center gap-1.5 text-slate-500">
              <Calendar className="w-3.5 h-3.5" />
              Created {new Date(poll.created_at).toLocaleDateString()}
            </span>

            {lastUpdated && (
              <span className="text-slate-500 ml-auto text-[11px] font-mono flex items-center gap-1">
                <RefreshCw className="w-3 h-3 text-emerald-400/80 animate-spin" style={{ animationDuration: '3s' }} />
                <span>Live Event: {lastUpdated.toLocaleTimeString()}</span>
              </span>
            )}
          </div>
        </div>

        {/* Notices */}
        {isClosed && (
          <div className="p-3.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-300 text-xs flex items-center gap-2.5">
            <Lock className="w-4 h-4 text-amber-400 shrink-0" />
            <span>This poll is closed. Voting has concluded, but real-time results remain visible.</span>
          </div>
        )}

        {hasVoted && (
          <div className="p-3.5 bg-sky-950/30 border border-sky-500/30 rounded-xl text-sky-300 text-xs flex items-center gap-2.5">
            <CheckCircle2 className="w-4 h-4 text-sky-400 shrink-0" />
            <span>Your vote is confirmed and recorded. Live results update automatically via Redis Pub/Sub.</span>
          </div>
        )}

        {voteError && (
          <div className="p-3.5 bg-rose-950/40 border border-rose-800/60 rounded-xl text-rose-300 text-xs flex items-center gap-2.5">
            <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />
            <span>{voteError}</span>
          </div>
        )}

        {/* View Toggle if hasn't voted yet */}
        {!hasVoted && !isClosed && (
          <div className="flex items-center justify-end pb-1">
            <button
              type="button"
              onClick={() => setViewResultsMode(!viewResultsMode)}
              className="text-xs text-sky-400 hover:text-sky-300 flex items-center gap-1.5 font-medium cursor-pointer"
            >
              <BarChart2 className="w-3.5 h-3.5" />
              <span>{viewResultsMode ? 'Switch to Voting Form' : 'Preview Live Results'}</span>
            </button>
          </div>
        )}

        {/* INTERFACE MODE 1: VOTING FORM (When user hasn't voted yet & wants to vote) */}
        {!showResults ? (
          <div className="space-y-4">
            <div className="space-y-2.5">
              {safeOptions.length === 0 ? (
                <div className="p-6 text-center border border-dashed border-slate-700 rounded-xl text-slate-400 text-xs">
                  This poll has no options yet.
                </div>
              ) : (
                safeOptions.map((option, idx) => {
                const isSelected = selectedOptionId === option.id;

                return (
                  <label
                    key={option.id}
                    id={`vote-option-${option.id}`}
                    className={`relative flex items-center justify-between p-4 rounded-xl border transition-all cursor-pointer select-none ${
                      isSelected
                        ? 'border-sky-500 bg-sky-950/30 shadow-md ring-1 ring-sky-500/50'
                        : 'border-slate-800 bg-slate-950/60 hover:border-slate-700 hover:bg-slate-900'
                    }`}
                  >
                    <div className="flex items-center gap-3.5">
                      <div
                        className={`w-5 h-5 rounded-full border flex items-center justify-center transition-colors ${
                          isSelected
                            ? 'border-sky-400 bg-sky-500 text-slate-950'
                            : 'border-slate-700 bg-slate-900'
                        }`}
                      >
                        {isSelected && <div className="w-2 h-2 rounded-full bg-slate-950" />}
                      </div>

                      <span className="text-sm font-medium text-slate-100">
                        {option.text}
                      </span>
                    </div>

                    <input
                      type="radio"
                      name={`poll-vote-${poll.id}`}
                      value={option.id}
                      checked={isSelected}
                      onChange={() => setSelectedOptionId(option.id)}
                      className="sr-only"
                    />

                    <span className="text-xs text-slate-500 font-mono">
                      #{idx + 1}
                    </span>
                  </label>
                );
              })
              )}
            </div>

            {/* Cast Vote Action Button */}
            <div className="pt-2">
              <button
                id="cast-vote-btn"
                type="button"
                onClick={handleCastVote}
                disabled={!selectedOptionId || isSubmittingVote}
                className="w-full py-3 px-4 rounded-xl bg-sky-500 hover:bg-sky-400 disabled:opacity-50 text-slate-950 font-bold text-sm shadow-lg shadow-sky-500/10 transition-all flex items-center justify-center gap-2 cursor-pointer"
              >
                {isSubmittingVote ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Transmitting Vote to Redis...</span>
                  </>
                ) : (
                  <>
                    <span>Cast Vote</span>
                  </>
                )}
              </button>
            </div>
          </div>
        ) : (
          /* INTERFACE MODE 2: LIVE RESULTS VIEW (With real SSE updates, animated bars, rankings) */
          <div className="space-y-3">
            {safeOptions.map((option) => {
              const percentage =
                poll.total_votes > 0
                  ? Math.round((option.vote_count / poll.total_votes) * 100)
                  : 0;
              const isVoted = votedOptionId === option.id;
              const isLeader = highestVoteCount > 0 && option.vote_count === highestVoteCount;
              const isPulsing = recentlyUpdatedOptionId === option.id;

              return (
                <div
                  key={option.id}
                  id={`option-result-${option.id}`}
                  className={`relative overflow-hidden rounded-xl border transition-all duration-300 ${
                    isPulsing
                      ? 'border-emerald-500 bg-emerald-950/20 ring-2 ring-emerald-500/40'
                      : isVoted
                      ? 'border-sky-500/80 bg-slate-900/90 shadow-md'
                      : 'border-slate-800/90 bg-slate-950/70'
                  }`}
                >
                  {/* Dynamic Progress Fill with CSS Transition */}
                  <div
                    className={`absolute top-0 bottom-0 left-0 transition-all duration-700 ease-out ${
                      isPulsing
                        ? 'bg-emerald-500/25'
                        : isVoted
                        ? 'bg-sky-500/20'
                        : isLeader
                        ? 'bg-sky-500/10'
                        : 'bg-slate-800/60'
                    }`}
                    style={{ width: `${percentage}%` }}
                  />

                  <div className="relative p-4 flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3 min-w-0">
                      <span className="font-semibold text-white text-sm truncate">
                        {option.text}
                      </span>

                      {isVoted && (
                        <span className="inline-flex items-center gap-1 text-[11px] font-bold text-sky-400 bg-sky-950/80 border border-sky-500/30 px-2.5 py-0.5 rounded-full shrink-0">
                          <CheckCircle2 className="w-3 h-3" />
                          Your Vote
                        </span>
                      )}

                      {isLeader && (
                        <span className="inline-flex items-center gap-1 text-[10px] font-bold text-amber-300 bg-amber-950/60 border border-amber-500/30 px-2 py-0.5 rounded-full shrink-0">
                          <Award className="w-3 h-3" />
                          Leader
                        </span>
                      )}
                    </div>

                    <div className="flex items-center gap-3 shrink-0 font-mono">
                      <div className="text-right">
                        <span className={`text-base font-bold transition-colors ${isPulsing ? 'text-emerald-400' : 'text-white'}`}>
                          {option.vote_count}
                        </span>
                        <span className="text-xs text-slate-400 ml-1.5 tabular-nums">
                          ({percentage}%)
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Owner Controls */}
        {isOwner && (
          <div className="pt-4 border-t border-slate-800/80 flex flex-wrap items-center justify-between gap-3">
            <span className="text-xs text-slate-500 font-medium">Owner Controls:</span>

            <div className="flex items-center gap-2.5">
              {!isClosed && (
                <button
                  id="close-poll-btn"
                  type="button"
                  onClick={() => {
                    if (!authToken) {
                      onRequireAuth?.();
                    } else {
                      setIsCloseModalOpen(true);
                    }
                  }}
                  className="px-3.5 py-2 text-xs font-semibold rounded-xl bg-amber-500/10 hover:bg-amber-500/20 text-amber-400 border border-amber-500/30 transition-colors cursor-pointer flex items-center gap-1.5"
                >
                  <Lock className="w-3.5 h-3.5 text-amber-400" />
                  <span>Close Poll</span>
                </button>
              )}

              <button
                id="delete-poll-btn"
                type="button"
                onClick={() => setIsDeleteModalOpen(true)}
                className="px-3.5 py-2 text-xs font-semibold rounded-xl bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/30 transition-colors cursor-pointer flex items-center gap-1.5"
              >
                <Trash2 className="w-3.5 h-3.5 text-rose-400" />
                <span>Delete Poll</span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Close Confirmation Modal */}
      {isOwner && (
        <ClosePollModal
          isOpen={isCloseModalOpen}
          pollId={poll.id}
          pollTitle={poll.title}
          onClose={() => setIsCloseModalOpen(false)}
          onClosed={() => {
            showToast('Poll voting concluded.', 'info', 'Poll Closed');
          }}
          onError={(msg) => {
            showToast(msg, 'error', 'Close Failed');
          }}
        />
      )}

      {/* Delete Confirmation Modal */}
      {isOwner && (
        <DeletePollModal
          isOpen={isDeleteModalOpen}
          pollId={poll.id}
          pollTitle={poll.title}
          onClose={() => setIsDeleteModalOpen(false)}
          onDeleted={(deletedId) => {
            showToast('Poll has been permanently deleted.', 'info', 'Poll Deleted');
            onPollDeleted ? onPollDeleted(deletedId) : onBack();
          }}
          onError={(msg) => {
            showToast(msg, 'error', 'Delete Failed');
          }}
        />
      )}
    </article>
  );
};
