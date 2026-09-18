import React from 'react';
import { Activity, BarChart3, Wifi, HardDrive } from 'lucide-react';
import { Poll } from '../types';

interface StatsBarProps {
  polls: Poll[];
  isHealthy: boolean;
  isRealtime?: boolean;
}

export const StatsBar: React.FC<StatsBarProps> = ({ polls, isHealthy, isRealtime = true }) => {
  const activeCount = polls.filter((p) => p.status === 'active').length;
  const closedCount = polls.filter((p) => p.status === 'closed').length;
  const totalVotesCount = polls.reduce((acc, p) => acc + p.total_votes, 0);
  const isLive = isHealthy && isRealtime;

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3 py-4">
      {/* Stat 1: Active Polls */}
      <div className="p-3.5 rounded-xl bg-slate-900/70 border border-slate-800/80 flex items-center gap-3">
        <div className="w-9 h-9 rounded-lg bg-sky-500/10 border border-sky-500/20 flex items-center justify-center text-sky-400 shrink-0">
          <BarChart3 className="w-4 h-4" />
        </div>
        <div>
          <div className="text-[11px] text-slate-400 font-medium">Active Polls</div>
          <div className="text-lg font-bold text-white font-mono tracking-tight">{activeCount}</div>
        </div>
      </div>

      {/* Stat 2: Total Votes Recorded */}
      <div className="p-3.5 rounded-xl bg-slate-900/70 border border-slate-800/80 flex items-center gap-3">
        <div className="w-9 h-9 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 shrink-0">
          <Activity className="w-4 h-4" />
        </div>
        <div>
          <div className="text-[11px] text-slate-400 font-medium">Total Votes Cast</div>
          <div className="text-lg font-bold text-white font-mono tracking-tight">{totalVotesCount}</div>
        </div>
      </div>

      {/* Stat 3: Realtime Transport */}
      <div className="p-3.5 rounded-xl bg-slate-900/70 border border-slate-800/80 flex items-center gap-3">
        <div className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 ${isLive ? 'bg-emerald-500/10 border border-emerald-500/20 text-emerald-400' : 'bg-rose-500/10 border border-rose-500/20 text-rose-400'}`}>
          <Wifi className="w-4 h-4" />
        </div>
        <div>
          <div className="text-[11px] text-slate-400 font-medium">Realtime Engine</div>
          <div className={`text-xs font-semibold font-mono tracking-tight flex items-center gap-1.5 ${isLive ? 'text-emerald-400' : 'text-rose-400'}`}>
            {isLive && <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping inline-block" />}
            <span>{isLive ? 'SSE Live Stream' : 'Disconnected'}</span>
          </div>
        </div>
      </div>

      {/* Stat 4: Polls Closed */}
      <div className="p-3.5 rounded-xl bg-slate-900/70 border border-slate-800/80 flex items-center gap-3">
        <div className="w-9 h-9 rounded-lg bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400 shrink-0">
          <HardDrive className="w-4 h-4" />
        </div>
        <div>
          <div className="text-[11px] text-slate-400 font-medium">Closed Polls</div>
          <div className="text-lg font-bold text-white font-mono tracking-tight">{closedCount}</div>
        </div>
      </div>
    </div>
  );
};
