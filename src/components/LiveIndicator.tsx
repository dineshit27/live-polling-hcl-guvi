import React from 'react';
import { RealtimeConnectionState } from '../types';
import { RefreshCw, WifiOff, Radio } from 'lucide-react';

interface LiveIndicatorProps {
  status: RealtimeConnectionState;
  reconnectAttempts?: number;
  onReconnect?: () => void;
  className?: string;
}

export const LiveIndicator: React.FC<LiveIndicatorProps> = ({
  status,
  reconnectAttempts = 0,
  onReconnect,
  className = '',
}) => {
  if (status === 'connected') {
    return (
      <div
        id="live-indicator-connected"
        role="status"
        aria-live="polite"
        className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/25 shadow-xs select-none ${className}`}
        title="Realtime SSE stream active via Redis Pub/Sub"
      >
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-60"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-400"></span>
        </span>
        <span className="font-semibold tracking-wide uppercase text-[10px]">Live Stream</span>
      </div>
    );
  }

  if (status === 'reconnecting') {
    return (
      <div
        id="live-indicator-reconnecting"
        role="status"
        aria-live="polite"
        className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/25 shadow-xs select-none ${className}`}
        title={`Attempting reconnection (${reconnectAttempts}/5)`}
      >
        <RefreshCw className="w-3 h-3 animate-spin text-amber-400" />
        <span className="text-[11px] font-medium">Reconnecting ({reconnectAttempts}/5)...</span>
      </div>
    );
  }

  if (status === 'connecting') {
    return (
      <div
        id="live-indicator-connecting"
        role="status"
        aria-live="polite"
        className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium bg-sky-500/10 text-sky-400 border border-sky-500/25 shadow-xs select-none ${className}`}
      >
        <span className="w-2 h-2 rounded-full bg-sky-400 animate-pulse"></span>
        <span className="text-[11px] font-medium">Connecting...</span>
      </div>
    );
  }

  // Disconnected state
  return (
    <div
      id="live-indicator-disconnected"
      role="status"
      aria-live="polite"
      className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-slate-900/90 text-slate-400 border border-slate-700 shadow-xs select-none ${className}`}
    >
      <WifiOff className="w-3 h-3 text-slate-500" />
      <span className="text-[11px]">Offline</span>
      {onReconnect && (
        <button
          type="button"
          onClick={onReconnect}
          className="ml-1 text-[11px] font-semibold text-sky-400 hover:text-sky-300 underline underline-offset-2 transition-colors cursor-pointer"
        >
          Reconnect
        </button>
      )}
    </div>
  );
};
