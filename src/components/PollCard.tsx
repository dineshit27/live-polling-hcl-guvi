import React, { useState } from 'react';
import { Poll } from '../types';
import { Users, Clock, ArrowUpRight, Share2, Check, Trash2, Lock, Flame } from 'lucide-react';

interface PollCardProps {
  poll: Poll;
  isOwner: boolean;
  onSelect: (pollId: string) => void;
  onDeleteClick?: (poll: Poll) => void;
  onCopyLink: (pollId: string) => void;
}

export const PollCard: React.FC<PollCardProps> = ({
  poll,
  isOwner,
  onSelect,
  onDeleteClick,
  onCopyLink,
}) => {
  const [copied, setCopied] = useState(false);
  const isClosed = poll.status === 'closed';

  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    onCopyLink(poll.id);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Find the leading option if there are votes
  const leadingOption =
    poll.total_votes > 0
      ? [...poll.options].sort((a, b) => b.vote_count - a.vote_count)[0]
      : null;

  return (
    <div
      id={`poll-card-${poll.id}`}
      onClick={() => onSelect(poll.id)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onSelect(poll.id);
        }
      }}
      className="group relative rounded-2xl bg-slate-900/80 hover:bg-slate-900 border border-slate-800 hover:border-sky-500/40 p-5 sm:p-6 transition-all duration-200 cursor-pointer shadow-md hover:shadow-xl hover:shadow-sky-500/5 focus-visible:border-sky-500"
    >
      {/* Top Meta Header */}
      <div className="flex items-center justify-between gap-3 mb-3">
        <div className="flex items-center gap-2 flex-wrap">
          {/* Status Badge */}
          <span
            className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[10px] font-bold tracking-wide uppercase ${
              isClosed
                ? 'bg-slate-800 text-slate-400 border border-slate-700'
                : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/25'
            }`}
          >
            <span
              className={`w-1.5 h-1.5 rounded-full ${
                isClosed ? 'bg-slate-500' : 'bg-emerald-400 animate-pulse'
              }`}
            />
            <span>{poll.status}</span>
          </span>

          {/* Owner badge */}
          {isOwner && (
            <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-sky-500/10 text-sky-400 border border-sky-500/20">
              Your Poll
            </span>
          )}

          {/* Hot/Active indicator if high votes */}
          {poll.total_votes >= 10 && !isClosed && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20">
              <Flame className="w-3 h-3 text-amber-400" />
              <span>Trending</span>
            </span>
          )}
        </div>

        {/* Action icons */}
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={handleCopy}
            title="Copy poll link"
            className="p-1.5 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Share2 className="w-3.5 h-3.5" />}
          </button>

          {isOwner && onDeleteClick && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onDeleteClick(poll);
              }}
              title="Delete poll"
              className="p-1.5 text-slate-400 hover:text-rose-400 rounded-lg hover:bg-rose-950/40 transition-colors cursor-pointer"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}

          <div className="p-1.5 text-slate-500 group-hover:text-sky-400 transition-colors ml-1">
            <ArrowUpRight className="w-4 h-4 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
          </div>
        </div>
      </div>

      {/* Poll Question Title */}
      <h3 className="text-base sm:text-lg font-bold text-white group-hover:text-sky-300 transition-colors tracking-tight line-clamp-2">
        {poll.title}
      </h3>

      {/* Description */}
      {poll.description && (
        <p className="text-xs text-slate-400 line-clamp-2 mt-1.5 leading-relaxed">
          {poll.description}
        </p>
      )}

      {/* Mini Options Preview Bar */}
      <div className="mt-4 pt-3 border-t border-slate-800/80">
        {poll.options.length > 0 && (
          <div className="space-y-1.5">
            {leadingOption && leadingOption.vote_count > 0 && (
              <div className="flex items-center justify-between text-[11px] text-slate-400">
                <span className="truncate max-w-[200px]">
                  Leading: <strong className="text-slate-200">{leadingOption.text}</strong>
                </span>
                <span className="font-mono text-sky-400 font-medium">
                  {Math.round((leadingOption.vote_count / poll.total_votes) * 100)}%
                </span>
              </div>
            )}

            {/* Micro stacked distribution bar */}
            <div className="h-1.5 w-full bg-slate-800 rounded-full overflow-hidden flex">
              {poll.total_votes > 0 ? (
                poll.options.map((opt, i) => {
                  const pct = (opt.vote_count / poll.total_votes) * 100;
                  if (pct === 0) return null;
                  const colors = ['bg-sky-400', 'bg-teal-400', 'bg-emerald-400', 'bg-purple-400', 'bg-amber-400'];
                  const color = colors[i % colors.length];
                  return (
                    <div
                      key={opt.id}
                      className={`h-full ${color}`}
                      style={{ width: `${pct}%` }}
                      title={`${opt.text}: ${opt.vote_count} votes`}
                    />
                  );
                })
              ) : (
                <div className="h-full w-full bg-slate-800/80" />
              )}
            </div>
          </div>
        )}

        {/* Footer Statistics */}
        <div className="flex items-center justify-between gap-4 mt-3 text-xs text-slate-400 font-mono">
          <div className="flex items-center gap-1.5">
            <Users className="w-3.5 h-3.5 text-sky-400" />
            <span className="text-slate-200 font-semibold">{poll.total_votes}</span>
            <span>{poll.total_votes === 1 ? 'vote' : 'votes'}</span>
          </div>

          <div className="flex items-center gap-3 text-[11px] text-slate-500">
            <span>{poll.options.length} options</span>
            <span className="flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {new Date(poll.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
