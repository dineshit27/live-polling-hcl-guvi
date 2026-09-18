import React, { useState } from 'react';
import { User, HealthCheckResponse } from '../types';
import {
  Plus,
  LogIn,
  LogOut,
  User as UserIcon,
  Activity,
  Menu,
  X,
  Sparkles,
} from 'lucide-react';

interface NavigationProps {
  currentUser: User | null;
  health: HealthCheckResponse | null;
  onOpenCreatePoll: () => void;
  onOpenAuth: () => void;
  onLogout: () => void;
  onOpenHealth: () => void;
  onHomeClick: () => void;
}

export const Navigation: React.FC<NavigationProps> = ({
  currentUser,
  health,
  onOpenCreatePoll,
  onOpenAuth,
  onLogout,
  onOpenHealth,
  onHomeClick,
}) => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const isHealthy = health?.status === 'healthy' || health?.status === 'ok';

  return (
    <header className="sticky top-0 z-40 w-full border-b border-slate-800/80 bg-slate-950/80 backdrop-blur-xl transition-all">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16 gap-4">
          
          {/* Brand Emblem & Logo */}
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={onHomeClick}
              className="flex items-center gap-2.5 text-left group cursor-pointer focus-visible:outline-none"
            >
              <div className="w-9 h-9 rounded-xl shadow-md shadow-sky-500/10 group-hover:shadow-sky-500/25 transition-all overflow-hidden flex-shrink-0">
                <img
                  src="/src/images/pulse-poll.jpg"
                  alt="PULSE logo"
                  className="w-full h-full object-cover"
                />
              </div>

              <div>
                <div className="flex items-center gap-1.5">
                  <span className="font-extrabold text-white text-base tracking-tight leading-none group-hover:text-sky-300 transition-colors">
                    PULSE
                  </span>
                  <span className="text-[10px] font-mono px-1.5 py-0.5 rounded-full bg-sky-500/10 text-sky-400 border border-sky-500/20 font-semibold uppercase">
                    Live
                  </span>
                </div>
                <span className="text-[11px] text-slate-500 font-medium block leading-tight">
                  Realtime Polling Engine
                </span>
              </div>
            </button>
          </div>

          {/* Desktop Right Navigation Controls */}
          <div className="hidden md:flex items-center gap-3">
            {/* System Architecture / Health Badge Button */}
            <button
              type="button"
              onClick={onOpenHealth}
              title="Click to inspect real Go, Redis, and MongoDB telemetry"
              className="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-mono bg-slate-900/80 hover:bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-300 transition-all cursor-pointer"
            >
              <span className="relative flex h-2 w-2">
                {isHealthy ? (
                  <>
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-60"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-400"></span>
                  </>
                ) : (
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-amber-400"></span>
                )}
              </span>
              <span>{isHealthy ? 'Go + Redis: Online' : 'Starting Services...'}</span>
            </button>

            {/* Create Poll CTA */}
            <button
              type="button"
              onClick={onOpenCreatePoll}
              className="px-3.5 py-1.5 rounded-xl bg-sky-500 hover:bg-sky-400 text-slate-950 font-bold text-xs shadow-md transition-all flex items-center gap-1.5 cursor-pointer"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>New Poll</span>
            </button>

            {/* Auth / Profile Area */}
            {currentUser ? (
              <div className="flex items-center gap-2 pl-2 border-l border-slate-800">
                <div className="flex items-center gap-2 px-2.5 py-1 rounded-xl bg-slate-900 border border-slate-800 text-xs text-slate-200">
                  <div className="w-5 h-5 rounded-full bg-sky-500/20 text-sky-400 flex items-center justify-center font-bold text-[10px]">
                    {((currentUser.name || currentUser.email || 'U').charAt(0) || 'U').toUpperCase()}
                  </div>
                  <span className="font-semibold text-xs max-w-[100px] truncate">
                    {currentUser.name || currentUser.email || 'User'}
                  </span>
                </div>

                <button
                  type="button"
                  onClick={onLogout}
                  title="Sign Out"
                  className="p-1.5 rounded-xl text-slate-400 hover:text-rose-400 hover:bg-rose-950/20 border border-transparent hover:border-rose-900/40 transition-colors cursor-pointer"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={onOpenAuth}
                className="px-3 py-1.5 rounded-xl bg-slate-900 hover:bg-slate-850 border border-slate-800 hover:border-slate-700 text-slate-200 hover:text-white text-xs font-semibold transition-colors flex items-center gap-1.5 cursor-pointer"
              >
                <LogIn className="w-3.5 h-3.5" />
                <span>Sign In</span>
              </button>
            )}
          </div>

          {/* Mobile Menu Toggle Button */}
          <div className="flex md:hidden items-center gap-2">
            <button
              type="button"
              onClick={onOpenCreatePoll}
              className="p-2 rounded-xl bg-sky-500 text-slate-950 font-bold cursor-pointer"
              title="Create Poll"
            >
              <Plus className="w-4 h-4" />
            </button>

            <button
              type="button"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="p-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 hover:text-white cursor-pointer"
              aria-label="Toggle mobile menu"
            >
              {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            </button>
          </div>

        </div>

        {/* Mobile Dropdown Menu */}
        {mobileMenuOpen && (
          <div className="md:hidden py-4 border-t border-slate-800 space-y-3">
            <button
              type="button"
              onClick={() => {
                onOpenHealth();
                setMobileMenuOpen(false);
              }}
              className="w-full flex items-center justify-between p-2.5 rounded-xl bg-slate-900 border border-slate-800 text-xs text-slate-300"
            >
              <span className="flex items-center gap-2">
                <Activity className="w-4 h-4 text-emerald-400" />
                <span>Infrastructure Telemetry</span>
              </span>
              <span className="font-mono text-emerald-400">{isHealthy ? 'Online' : 'Pending'}</span>
            </button>

            {currentUser ? (
              <div className="p-3 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-6 h-6 rounded-full bg-sky-500/20 text-sky-400 flex items-center justify-center font-bold text-xs">
                    {((currentUser.name || currentUser.email || 'U').charAt(0) || 'U').toUpperCase()}
                  </div>
                  <div>
                    <div className="text-xs font-bold text-white">
                      {currentUser.name || currentUser.email || 'User'}
                    </div>
                    {currentUser.email && (
                      <div className="text-[11px] text-slate-400">{currentUser.email}</div>
                    )}
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => {
                    onLogout();
                    setMobileMenuOpen(false);
                  }}
                  className="px-3 py-1.5 rounded-lg bg-rose-950/40 text-rose-400 border border-rose-800/40 text-xs font-semibold cursor-pointer"
                >
                  Sign Out
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => {
                  onOpenAuth();
                  setMobileMenuOpen(false);
                }}
                className="w-full py-2.5 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-semibold flex items-center justify-center gap-2 cursor-pointer"
              >
                <LogIn className="w-4 h-4" />
                <span>Sign In / Register</span>
              </button>
            )}
          </div>
        )}
      </div>
    </header>
  );
};
