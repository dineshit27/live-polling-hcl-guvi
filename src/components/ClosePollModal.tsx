import React, { useState, useEffect } from 'react';
import { Lock, X, Loader2, AlertCircle } from 'lucide-react';
import { api, ApiError } from '../services/api';

interface ClosePollModalProps {
  isOpen: boolean;
  pollId: string;
  pollTitle: string;
  onClose: () => void;
  onClosed: () => void;
  onError: (message: string) => void;
}

export const ClosePollModal: React.FC<ClosePollModalProps> = ({
  isOpen,
  pollId,
  pollTitle,
  onClose,
  onClosed,
  onError,
}) => {
  const [isClosing, setIsClosing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape' && !isClosing) {
          onClose();
        }
      };
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [isOpen, isClosing, onClose]);

  if (!isOpen) return null;

  const handleClose = async () => {
    setIsClosing(true);
    setError(null);
    try {
      await api.closePoll(pollId);
      onClosed();
      onClose();
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : 'Failed to close poll';
      setError(msg);
      onError(msg);
    } finally {
      setIsClosing(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="close-dialog-title"
      className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 overflow-y-auto"
      onClick={onClose}
    >
      <div
        id="close-poll-modal"
        className="bg-slate-900 border border-slate-800 rounded-2xl max-w-md w-full p-6 sm:p-8 relative shadow-2xl space-y-5 my-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-2.5 text-amber-400">
            <div className="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/25 flex items-center justify-center">
              <Lock className="w-4 h-4 text-amber-400" />
            </div>
            <h3 id="close-dialog-title" className="text-base font-bold text-white tracking-tight">
              Close Poll Voting
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isClosing}
            aria-label="Close dialog"
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {error && (
          <div className="p-3 bg-rose-950/40 border border-rose-800/60 rounded-xl text-rose-300 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <div className="space-y-3 text-xs text-slate-300 leading-relaxed">
          <p>
            Are you sure you want to conclude voting on{' '}
            <strong className="text-white font-semibold">"{pollTitle}"</strong>?
          </p>
          <div className="p-3 rounded-xl bg-slate-950 border border-slate-800/80 text-slate-400">
            Once closed, no further votes can be cast by participants. Current vote counts and real-time final results will remain permanently accessible.
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-800/80">
          <button
            type="button"
            onClick={onClose}
            disabled={isClosing}
            className="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white rounded-xl transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            id="confirm-close-poll-btn"
            type="button"
            onClick={handleClose}
            disabled={isClosing}
            className="px-4 py-2 text-xs font-bold rounded-xl bg-amber-500 hover:bg-amber-400 text-slate-950 shadow-md transition-all flex items-center gap-1.5 cursor-pointer"
          >
            {isClosing ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                <span>Closing...</span>
              </>
            ) : (
              <>
                <Lock className="w-3.5 h-3.5" />
                <span>Confirm Close</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
