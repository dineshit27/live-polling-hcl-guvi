import React, { useState, useEffect, useRef } from 'react';
import { X, Plus, Trash2, Loader2, Sparkles, Radio, Eye, Check } from 'lucide-react';
import { api, ApiError } from '../services/api';
import { useToast } from './Toast';

interface CreatePollModalProps {
  isOpen: boolean;
  authToken: string | null;
  onClose: () => void;
  onSuccess: (newPollId: string) => void;
  onRequireAuth: () => void;
}

export const CreatePollModal: React.FC<CreatePollModalProps> = ({
  isOpen,
  authToken,
  onClose,
  onSuccess,
  onRequireAuth,
}) => {
  const { showToast } = useToast();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [options, setOptions] = useState<string[]>(['', '', '']);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'editor' | 'preview'>('editor');

  const titleInputRef = useRef<HTMLInputElement | null>(null);

  // Focus title input on open and handle ESC key
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => {
        titleInputRef.current?.focus();
      }, 100);

      const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape' && !isSubmitting) {
          onClose();
        }
      };
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [isOpen, isSubmitting, onClose]);

  if (!isOpen) return null;

  const handleOptionChange = (index: number, val: string) => {
    const updated = [...options];
    updated[index] = val;
    setOptions(updated);
    if (error) setError(null);
  };

  const handleAddOption = () => {
    if (options.length < 10) {
      setOptions([...options, '']);
    }
  };

  const handleRemoveOption = (index: number) => {
    if (options.length > 2) {
      setOptions(options.filter((_, i) => i !== index));
    }
  };

  const handleOptionKeyDown = (e: React.KeyboardEvent, index: number) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (index === options.length - 1 && options.length < 10) {
        handleAddOption();
      }
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!authToken) {
      onRequireAuth();
      return;
    }

    const trimmedTitle = title.trim();
    if (trimmedTitle.length < 5) {
      setError('Poll question must be at least 5 characters long.');
      return;
    }

    const cleanOptions = options.map((o) => o.trim()).filter(Boolean);
    if (cleanOptions.length < 2) {
      setError('At least two non-empty options are required to publish a poll.');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const newPoll = await api.createPoll({
        title: trimmedTitle,
        description: description.trim() || undefined,
        options: cleanOptions,
      });

      showToast('Your poll is live and ready for real-time voting!', 'success', 'Poll Published');
      onSuccess(newPoll.id);
      setTitle('');
      setDescription('');
      setOptions(['', '', '']);
      onClose();
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : 'Network error while creating poll.';
      setError(msg);
      showToast(msg, 'error', 'Creation Failed');
    } finally {
      setIsSubmitting(false);
    }
  };

  const previewOptions = options.filter((o) => o.trim().length > 0);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-poll-title"
      className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-3 sm:p-6 overflow-y-auto"
    >
      <div
        className="bg-slate-900 border border-slate-800 rounded-2xl max-w-4xl w-full p-5 sm:p-8 relative shadow-2xl space-y-6 my-auto"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-sky-500/10 border border-sky-500/25 flex items-center justify-center text-sky-400">
              <Sparkles className="w-4 h-4" />
            </div>
            <div>
              <h2 id="create-poll-title" className="text-lg font-bold text-white tracking-tight">
                Create Live Poll
              </h2>
              <p className="text-xs text-slate-400">
                Configure your question, options, and observe the live preview.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Mobile Tab Toggle */}
            <div className="flex lg:hidden bg-slate-950 p-0.5 rounded-lg border border-slate-800 text-xs">
              <button
                type="button"
                onClick={() => setActiveTab('editor')}
                className={`px-2.5 py-1 rounded-md transition-colors ${
                  activeTab === 'editor' ? 'bg-slate-800 text-white font-medium' : 'text-slate-400'
                }`}
              >
                Editor
              </button>
              <button
                type="button"
                onClick={() => setActiveTab('preview')}
                className={`px-2.5 py-1 rounded-md transition-colors ${
                  activeTab === 'preview' ? 'bg-slate-800 text-white font-medium' : 'text-slate-400'
                }`}
              >
                Preview
              </button>
            </div>

            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              aria-label="Close modal"
              className="text-slate-400 hover:text-white p-1.5 rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {error && (
          <div className="p-3 bg-rose-950/40 border border-rose-800/60 rounded-xl text-rose-300 text-xs flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* 2-Column Responsive Layout: Left Editor, Right Live Preview */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 items-start">
          
          {/* Left Column: Form Editor */}
          <form
            onSubmit={handleSubmit}
            className={`space-y-4 lg:col-span-7 ${
              activeTab === 'editor' ? 'block' : 'hidden lg:block'
            }`}
          >
            {/* Question Input */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-semibold text-slate-200">
                  Question <span className="text-sky-400">*</span>
                </label>
                <span className="text-[11px] text-slate-500 font-mono">
                  {title.length}/150
                </span>
              </div>
              <input
                ref={titleInputRef}
                type="text"
                required
                maxLength={150}
                placeholder="e.g. Which programming language do you rely on most?"
                value={title}
                onChange={(e) => {
                  setTitle(e.target.value);
                  if (error) setError(null);
                }}
                className="w-full px-3.5 py-2.5 text-sm bg-slate-950 border border-slate-800 rounded-xl text-white placeholder:text-slate-600 focus:border-sky-500 focus:outline-none transition-colors"
              />
            </div>

            {/* Description Input */}
            <div>
              <label className="block text-xs font-semibold text-slate-200 mb-1.5">
                Description <span className="text-slate-500 font-normal">(optional context)</span>
              </label>
              <textarea
                rows={2}
                maxLength={300}
                placeholder="Provide helpful context or guidelines for participants..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full px-3.5 py-2 text-xs bg-slate-950 border border-slate-800 rounded-xl text-white placeholder:text-slate-600 focus:border-sky-500 focus:outline-none transition-colors resize-none leading-relaxed"
              />
            </div>

            {/* Dynamic Options List */}
            <div className="space-y-2.5 pt-1">
              <div className="flex items-center justify-between">
                <label className="text-xs font-semibold text-slate-200">
                  Poll Options <span className="text-sky-400">*</span>
                  <span className="text-[11px] font-normal text-slate-500 ml-1.5 font-mono">
                    ({options.length}/10)
                  </span>
                </label>
                {options.length < 10 && (
                  <button
                    type="button"
                    onClick={handleAddOption}
                    className="text-xs font-semibold text-sky-400 hover:text-sky-300 flex items-center gap-1 cursor-pointer transition-colors"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>Add Option</span>
                  </button>
                )}
              </div>

              <div className="space-y-2 max-h-56 overflow-y-auto pr-1">
                {options.map((opt, idx) => (
                  <div key={idx} className="flex items-center gap-2 group">
                    <span className="w-6 text-[11px] font-mono text-slate-500 text-center shrink-0">
                      {String(idx + 1).padStart(2, '0')}
                    </span>
                    <input
                      type="text"
                      required
                      placeholder={`Option ${idx + 1}`}
                      value={opt}
                      onChange={(e) => handleOptionChange(idx, e.target.value)}
                      onKeyDown={(e) => handleOptionKeyDown(e, idx)}
                      className="flex-1 px-3 py-2 text-xs bg-slate-950 border border-slate-800 rounded-lg text-white placeholder:text-slate-600 focus:border-sky-500 focus:outline-none transition-colors"
                    />
                    {options.length > 2 && (
                      <button
                        type="button"
                        onClick={() => handleRemoveOption(idx)}
                        className="p-1.5 text-slate-500 hover:text-rose-400 rounded-md hover:bg-rose-950/20 transition-colors cursor-pointer shrink-0"
                        title="Remove this option"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
              <button
                type="button"
                onClick={onClose}
                disabled={isSubmitting}
                className="px-4 py-2 text-xs font-medium text-slate-400 hover:text-white rounded-lg transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isSubmitting}
                className="px-5 py-2.5 text-xs font-semibold rounded-xl bg-sky-500 hover:bg-sky-400 disabled:opacity-50 text-slate-950 shadow-md transition-all flex items-center gap-2 cursor-pointer"
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Publishing...</span>
                  </>
                ) : (
                  <>
                    <span>Publish Poll</span>
                  </>
                )}
              </button>
            </div>
          </form>

          {/* Right Column: Live Interactive Preview */}
          <div
            className={`lg:col-span-5 bg-slate-950/80 border border-slate-800 rounded-2xl p-5 space-y-4 ${
              activeTab === 'preview' ? 'block' : 'hidden lg:block'
            }`}
          >
            <div className="flex items-center justify-between pb-3 border-b border-slate-800">
              <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                <Eye className="w-3.5 h-3.5 text-sky-400" />
                <span>Live Voter Preview</span>
              </div>
              <span className="text-[10px] font-mono text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                ACTIVE
              </span>
            </div>

            {/* Preview Card */}
            <div className="space-y-4">
              <div>
                <h3 className="text-base font-bold text-white tracking-tight">
                  {title.trim() ? title : 'Your poll question will appear here...'}
                </h3>
                {description.trim() && (
                  <p className="text-xs text-slate-400 mt-1 leading-relaxed">{description}</p>
                )}
              </div>

              {/* Options Preview */}
              <div className="space-y-2">
                {previewOptions.length > 0 ? (
                  previewOptions.map((optText, i) => (
                    <div
                      key={i}
                      className="p-3 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between text-xs"
                    >
                      <div className="flex items-center gap-2.5">
                        <span className="w-3.5 h-3.5 rounded-full border border-slate-700 flex items-center justify-center">
                          <span className="w-1.5 h-1.5 rounded-full bg-transparent" />
                        </span>
                        <span className="text-slate-200">{optText}</span>
                      </div>
                      <span className="text-slate-500 font-mono text-[11px]">0%</span>
                    </div>
                  ))
                ) : (
                  <div className="p-6 text-center border border-dashed border-slate-800 rounded-xl text-slate-500 text-xs">
                    Options you type on the left will render here live.
                  </div>
                )}
              </div>

              <div className="pt-2 text-center">
                <div className="w-full py-2 rounded-xl bg-slate-800/80 text-slate-400 text-xs font-semibold text-center select-none">
                  Vote Button (Active when published)
                </div>
              </div>
            </div>
          </div>

        </div>
      </div>
    </div>
  );
};
