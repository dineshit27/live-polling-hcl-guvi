import React, { useState, useEffect } from 'react';
import { AlertTriangle, Trash2, X, Loader2 } from 'lucide-react';
import { api, ApiError } from '../services/api';

interface DeletePollModalProps {
  isOpen: boolean;
  pollId: string;
  pollTitle: string;
  onClose: () => void;
  onDeleted: (pollId: string) => void;
  onError: (message: string) => void;
}

export const DeletePollModal: React.FC<DeletePollModalProps> = ({
  isOpen,
  pollId,
  pollTitle,
  onClose,
  onDeleted,
  onError,
}) => {
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape' && !isDeleting) {
          onClose();
        }
      };
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [isOpen, isDeleting, onClose]);

  if (!isOpen) return null;

  const handleDelete = async () => {
    setIsDeleting(true);
    setError(null);
    try {
      await api.deletePoll(pollId);
      onDeleted(pollId);
      onClose();
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : 'Failed to delete poll';
      setError(msg);
      onError(msg);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-dialog-title"
      className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 overflow-y-auto"
      onClick={onClose}
    >
      <div
        id="delete-poll-modal"
        className="bg-slate-900 border border-slate-800 rounded-2xl max-w-md w-full p-6 sm:p-8 relative shadow-2xl space-y-5 my-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-2.5 text-rose-400">
            <div className="w-8 h-8 rounded-lg bg-rose-500/10 border border-rose-500/25 flex items-center justify-center">
              <AlertTriangle className="w-4 h-4 text-rose-400" />
            </div>
            <h3 id="delete-dialog-title" className="text-base font-bold text-white tracking-tight">
              Delete Poll Permanently
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isDeleting}
            aria-label="Close dialog"
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {error && (
          <div className="p-3 bg-rose-950/40 border border-rose-800/60 rounded-xl text-rose-300 text-xs">
            {error}
          </div>
        )}

        <div className="space-y-3 text-xs text-slate-300 leading-relaxed">
          <p>
            Are you sure you want to permanently delete{' '}
            <strong className="text-white font-semibold">"{pollTitle}"</strong>?
          </p>
          <div className="p-3 rounded-xl bg-slate-950 border border-slate-800/80 text-slate-400">
            This action is irreversible. All recorded votes, voter tracking indices in MongoDB, and the associated Redis Pub/Sub stream will be terminated.
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-800/80">
          <button
            type="button"
            onClick={onClose}
            disabled={isDeleting}
            className="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white rounded-xl transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            id="confirm-delete-poll-btn"
            type="button"
            onClick={handleDelete}
            disabled={isDeleting}
            className="px-4 py-2 text-xs font-bold rounded-xl bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white shadow-md transition-all flex items-center gap-1.5 cursor-pointer"
          >
            {isDeleting ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                <span>Deleting...</span>
              </>
            ) : (
              <>
                <Trash2 className="w-3.5 h-3.5" />
                <span>Delete Poll</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
