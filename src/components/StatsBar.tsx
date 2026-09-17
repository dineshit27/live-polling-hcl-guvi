import React from 'react';
import { Activity, BarChart3, Database, Radio } from 'lucide-react';
import { Poll } from '../types';

interface StatsBarProps {
  polls: Poll[];
  isHealthy: boolean;
}

export const StatsBar: React.FC<StatsBarProps> = ({ polls, isHealthy }) => {
  const activeCount = polls.filter((p) => p.status === 'active').length;
  const totalVotesCount = polls.reduce((acc, p) => acc + p.total_votes, 0);

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

      {/* Stat 3: Pub/Sub Broker */}
      <div className="p-3.5 rounded-xl bg-slate-900/70 border border-slate-800/80 flex items-center gap-3">
        <div className="w-9 h-9 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400 shrink-0">
          <Radio className="w-4 h-4" />
        </div>
        <div>
          <div className="text-[11px] text-slate-400 font-medium">Realtime Bus</div>
          <div className="text-xs font-semibold text-slate-200 font-mono tracking-tight">Redis Pub/Sub</div>
        </div>
      </div>

      {/* Stat 4: Data Engine */}
      <div className="p-3.5 rounded-xl bg-slate-900/70 border border-slate-800/80 flex items-center gap-3">
        <div className="w-9 h-9 rounded-lg bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400 shrink-0">
          <Database className="w-4 h-4" />
        </div>
        <div>
          <div className="text-[11px] text-slate-400 font-medium">Persistence</div>
          <div className="text-xs font-semibold text-slate-200 font-mono tracking-tight">MongoDB 7.0</div>
        </div>
      </div>
    </div>
  );
};
