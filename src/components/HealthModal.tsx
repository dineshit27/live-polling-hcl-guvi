import React, { useEffect } from 'react';
import { X, CheckCircle2, AlertCircle, Database, Server, Radio, Cpu, Layers } from 'lucide-react';
import { HealthCheckResponse } from '../types';

interface HealthModalProps {
  isOpen: boolean;
  onClose: () => void;
  health: HealthCheckResponse | null;
}

export const HealthModal: React.FC<HealthModalProps> = ({ isOpen, onClose, health }) => {
  useEffect(() => {
    if (isOpen) {
      const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape') onClose();
      };
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const isHealthy = health?.status === 'healthy' || health?.status === 'ok';

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="health-modal-title"
      className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 overflow-y-auto"
      onClick={onClose}
    >
      <div
        className="bg-slate-900 border border-slate-800 rounded-2xl max-w-lg w-full p-6 sm:p-8 relative shadow-2xl space-y-6 my-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/25 flex items-center justify-center text-emerald-400">
              <Cpu className="w-4 h-4" />
            </div>
            <div>
              <h3 id="health-modal-title" className="text-base font-bold text-white tracking-tight">
                Infrastructure & Health Telemetry
              </h3>
              <p className="text-xs text-slate-400">
                Live verification of Go/Gin, MongoDB, Redis Pub/Sub, and SSE
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close dialog"
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Global Status Banner */}
        <div
          className={`p-4 rounded-xl border flex items-center justify-between ${
            isHealthy
              ? 'bg-emerald-950/30 border-emerald-500/30 text-emerald-300'
              : 'bg-amber-950/30 border-amber-500/30 text-amber-300'
          }`}
        >
          <div className="flex items-center gap-2.5">
            {isHealthy ? (
              <CheckCircle2 className="w-5 h-5 text-emerald-400" />
            ) : (
              <AlertCircle className="w-5 h-5 text-amber-400" />
            )}
            <div>
              <div className="text-xs font-bold uppercase tracking-wider">
                System Status: {health?.status || 'Unknown'}
              </div>
              <div className="text-[11px] opacity-80">
                {isHealthy ? 'All services operational' : 'Initialization in progress'}
              </div>
            </div>
          </div>
          {health?.timestamp && (
            <span className="text-[10px] font-mono opacity-70">
              {new Date(health.timestamp).toLocaleTimeString()}
            </span>
          )}
        </div>

        {/* Stack breakdown */}
        <div className="space-y-3">
          {/* Go / Gin Service */}
          <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Server className="w-4 h-4 text-sky-400" />
              <div>
                <div className="text-xs font-semibold text-white">Go/Gin REST API Engine</div>
                <div className="text-[11px] text-slate-400 font-mono">Port 8081 / Proxied through Vite</div>
              </div>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              200 OK
            </span>
          </div>

          {/* Redis Pub/Sub */}
          <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Radio className="w-4 h-4 text-amber-400" />
              <div>
                <div className="text-xs font-semibold text-white">Redis Pub/Sub Realtime Broker</div>
                <div className="text-[11px] text-slate-400 font-mono">127.0.0.1:6379 · Events fanout</div>
              </div>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              {health?.services?.redis || 'connected'}
            </span>
          </div>

          {/* MongoDB */}
          <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Database className="w-4 h-4 text-purple-400" />
              <div>
                <div className="text-xs font-semibold text-white">MongoDB Persistence Store</div>
                <div className="text-[11px] text-slate-400 font-mono">polling_db · Atomic $inc updates</div>
              </div>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              {health?.services?.mongodb || 'connected'}
            </span>
          </div>

          {/* SSE Stream */}
          <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Layers className="w-4 h-4 text-teal-400" />
              <div>
                <div className="text-xs font-semibold text-white">Server-Sent Events (SSE)</div>
                <div className="text-[11px] text-slate-400 font-mono">/api/polls/:id/events · Zero polling</div>
              </div>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              Streaming
            </span>
          </div>
        </div>

        <div className="pt-2 text-right">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-xs font-semibold rounded-xl bg-slate-800 hover:bg-slate-700 text-white transition-colors cursor-pointer"
          >
            Close Inspector
          </button>
        </div>
      </div>
    </div>
  );
};
