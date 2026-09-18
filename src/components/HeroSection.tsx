import React, { useState, useEffect } from 'react';
import { Plus, ArrowRight, CheckCircle2, Users, Flame } from 'lucide-react';

interface HeroSectionProps {
  onCreatePollClick: () => void;
  onExploreClick: () => void;
}

interface DemoOption {
  id: string;
  text: string;
  votes: number;
}

export const HeroSection: React.FC<HeroSectionProps> = ({
  onCreatePollClick,
  onExploreClick,
}) => {
  // Live interactive demo poll state inside the hero
  const [demoOptions, setDemoOptions] = useState<DemoOption[]>([
    { id: '1', text: 'AI Coding Assistant', votes: 48 },
    { id: '2', text: 'Realtime Multi-User Whiteboard', votes: 34 },
    { id: '3', text: 'High-Throughput Event Streamer', votes: 62 },
    { id: '4', text: 'Telemetry Observability Hub', votes: 21 },
  ]);
  const [votedOptionId, setVotedOptionId] = useState<string | null>(null);
  const [lastUpdatedOption, setLastUpdatedOption] = useState<string | null>(null);

  // Calculate totals
  const totalVotes = demoOptions.reduce((acc, opt) => acc + opt.votes, 0);

  // Subtle live pulse: periodically increment a vote every 6 seconds to show the live engine feeling
  useEffect(() => {
    const timer = setInterval(() => {
      setDemoOptions((prev) => {
        const randomIndex = Math.floor(Math.random() * prev.length);
        const updated = [...prev];
        const target = updated[randomIndex];
        updated[randomIndex] = {
          ...target,
          votes: target.votes + 1,
        };
        setLastUpdatedOption(target.id);
        return updated;
      });

      // Clear highlight after 1.2s
      setTimeout(() => {
        setLastUpdatedOption(null);
      }, 1200);
    }, 6000);

    return () => clearInterval(timer);
  }, []);

  const handleDemoVote = (id: string) => {
    if (votedOptionId) return;
    setVotedOptionId(id);
    setLastUpdatedOption(id);
    setDemoOptions((prev) =>
      prev.map((opt) => (opt.id === id ? { ...opt, votes: opt.votes + 1 } : opt))
    );
    setTimeout(() => setLastUpdatedOption(null), 1200);
  };

  return (
    <section className="relative pt-6 pb-12 sm:pt-10 sm:pb-16 lg:pb-20 overflow-hidden">
      {/* Subtle background ambient mesh */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full max-w-7xl h-96 pointer-events-none opacity-20 overflow-hidden">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-sky-600/30 rounded-full blur-3xl" />
        <div className="absolute top-1/3 right-1/4 w-80 h-80 bg-emerald-600/20 rounded-full blur-3xl" />
      </div>

      <div className="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-14 lg:gap-20 items-center">
          
          {/* Left Column: Product Value & Direct Copy */}
          <div className="lg:col-span-6 space-y-6 text-left">
            {/* Tech badge */}
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium bg-slate-900 border border-slate-800 text-slate-300 shadow-xs">
              <span className="flex h-2 w-2 relative">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-400"></span>
              </span>
              <span className="text-slate-400">Engine:</span>
              <span className="font-semibold text-white tracking-wide">Go · Redis Pub/Sub · SSE</span>
            </div>

            {/* Main Editorial Headline */}
            <div className="space-y-2">
              <h1 className="text-3xl sm:text-4xl lg:text-5xl font-extrabold text-white tracking-tight leading-[1.1]">
                Create polls. <br />
                Share them. <br />
                <span className="text-transparent bg-clip-text bg-gradient-to-r from-sky-400 via-teal-300 to-emerald-400">
                  Watch votes live.
                </span>
              </h1>
              <p className="text-sm sm:text-base text-slate-400 leading-relaxed max-w-lg pt-2">
                Real-time polling infrastructure engineered for instant feedback. Create shareable live polls
                with frictionless online voting, where every vote publishes through Redis channels and streams
                live results directly to connected browsers via SSE with sub-second delivery.
              </p>
            </div>

            {/* Action Buttons */}
            <div className="flex flex-wrap items-center gap-3.5 pt-2">
              <button
                id="hero-create-btn"
                type="button"
                onClick={onCreatePollClick}
                className="px-5 py-3 rounded-xl bg-sky-500 hover:bg-sky-400 text-slate-950 font-semibold text-sm shadow-md hover:shadow-sky-500/20 transition-all flex items-center gap-2 cursor-pointer group"
              >
                <Plus className="w-4 h-4 transition-transform group-hover:rotate-90 duration-200" />
                <span>Create a Poll</span>
              </button>

              <button
                id="hero-explore-btn"
                type="button"
                onClick={onExploreClick}
                className="px-4 py-3 rounded-xl bg-slate-900/90 hover:bg-slate-800 text-slate-200 hover:text-white border border-slate-800 hover:border-slate-700 font-medium text-sm transition-colors flex items-center gap-1.5 cursor-pointer"
              >
                <span>Explore Live Polls</span>
                <ArrowRight className="w-4 h-4 text-slate-400" />
              </button>
            </div>

            {/* Proof Points Strip */}
            <div className="grid grid-cols-3 gap-4 pt-4 border-t border-slate-800/80">
              <div>
                <div className="text-xs text-slate-500 font-medium">Latency</div>
                <div className="text-sm font-semibold text-slate-200 font-mono tracking-tight">&lt; 5ms Redis</div>
              </div>
              <div>
                <div className="text-xs text-slate-500 font-medium">Delivery</div>
                <div className="text-sm font-semibold text-slate-200 font-mono tracking-tight">Pure SSE Stream</div>
              </div>
              <div>
                <div className="text-xs text-slate-500 font-medium">Reliability</div>
                <div className="text-sm font-semibold text-slate-200 font-mono tracking-tight">Atomic $inc</div>
              </div>
            </div>
          </div>

          {/* Right Column: Live Interactive Poll Demonstration Card with Animated Glow Border */}
          <div className="lg:col-span-6 lg:pl-8 mt-4 lg:mt-0">
            {/* Glow border wrapper with capped width */}
            <div className="relative max-w-[420px] mx-auto lg:ml-auto p-[1.5px] rounded-2xl bg-gradient-to-br from-sky-500/50 via-teal-400/30 to-emerald-500/50 hero-poll-glow">
              <div className="relative rounded-2xl bg-slate-900/95 shadow-2xl p-4 sm:p-5 backdrop-blur-xl">
                {/* Subtle inner rim glow */}
                <div className="pointer-events-none absolute inset-0 rounded-2xl ring-1 ring-inset ring-sky-400/10" />

                {/* Header inside card */}
                <div className="flex items-center justify-between pb-2.5 border-b border-slate-800/80">
                  <div className="flex items-center gap-2">
                    <div className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/25">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping" />
                      <span>LIVE DEMO</span>
                    </div>
                    <span className="text-[10px] text-slate-400 font-mono">poll:demo:events</span>
                  </div>
                  <div className="flex items-center gap-1 text-[11px] text-slate-400 font-mono">
                    <Users className="w-3 h-3 text-sky-400" />
                    <span className="font-semibold text-white">{totalVotes}</span> votes
                  </div>
                </div>

                {/* Sample Question */}
                <div className="py-3">
                  <h3 className="text-sm sm:text-base font-bold text-white tracking-tight leading-snug">
                    What infrastructure component should we build next?
                  </h3>
                  <p className="text-[11px] text-slate-400 mt-1">
                    {votedOptionId
                      ? 'Vote recorded! Realtime updates simulate incoming votes.'
                      : 'Click an option to test instant vote broadcasting:'}
                  </p>
                </div>

                {/* Options list with animated bars */}
                <div className="space-y-2">
                  {demoOptions.map((opt) => {
                    const percentage = Math.round((opt.votes / totalVotes) * 100);
                    const isVoted = votedOptionId === opt.id;
                    const isHighlighted = lastUpdatedOption === opt.id;

                    return (
                      <button
                        key={opt.id}
                        type="button"
                        onClick={() => handleDemoVote(opt.id)}
                        className={`w-full text-left relative overflow-hidden rounded-xl border py-2 px-3 transition-all cursor-pointer ${
                          isVoted
                            ? 'border-sky-500/80 bg-sky-950/20'
                            : isHighlighted
                            ? 'border-emerald-500/60 bg-slate-800/90'
                            : 'border-slate-800/90 bg-slate-950/60 hover:border-slate-700 hover:bg-slate-900'
                        }`}
                      >
                        {/* Animated Progress Fill */}
                        <div
                          className={`absolute top-0 bottom-0 left-0 transition-all duration-500 ease-out ${
                            isVoted
                              ? 'bg-sky-500/20'
                              : isHighlighted
                              ? 'bg-emerald-500/20'
                              : 'bg-slate-800/50'
                          }`}
                          style={{ width: `${percentage}%` }}
                        />

                        <div className="relative flex items-center justify-between gap-2.5">
                          <div className="flex items-center gap-2 min-w-0">
                            <span
                              className={`w-3.5 h-3.5 rounded-full border flex items-center justify-center shrink-0 transition-colors ${
                                isVoted
                                  ? 'border-sky-400 bg-sky-500 text-slate-950'
                                  : 'border-slate-700 bg-slate-900'
                              }`}
                            >
                              {isVoted && <CheckCircle2 className="w-3 h-3" />}
                            </span>
                            <span className="text-xs font-medium text-slate-200 truncate">
                              {opt.text}
                            </span>
                          </div>

                          <div className="flex items-center gap-1.5 shrink-0 font-mono text-xs">
                            <span className={`font-semibold transition-colors ${isHighlighted ? 'text-emerald-400' : 'text-white'}`}>
                              {opt.votes}
                            </span>
                            <span className="text-slate-400 text-[10px] tabular-nums">
                              ({percentage}%)
                            </span>
                          </div>
                        </div>
                      </button>
                    );
                  })}
                </div>

                {/* Card Footer */}
                <div className="mt-3 pt-2.5 border-t border-slate-800/70 flex items-center justify-between text-[10px] text-slate-500">
                  <span className="flex items-center gap-1">
                    <Flame className="w-3 h-3 text-amber-400" />
                    <span>Realtime feedback without reload</span>
                  </span>
                  <span className="text-slate-400">Zero polling</span>
                </div>
              </div>
            </div>
          </div>

        </div>
      </div>
    </section>
  );
};
