import React, { useCallback, useEffect, useState, useRef } from 'react';
import { Poll, User, HealthCheckResponse } from './types';
import { LivePollView } from './components/LivePollView';
import { PollList } from './components/PollList';
import { HeroSection } from './components/HeroSection';
import { Navigation } from './components/Navigation';
import { CreatePollModal } from './components/CreatePollModal';
import { AuthModal } from './components/AuthModal';
import { DeletePollModal } from './components/DeletePollModal';
import { HealthModal } from './components/HealthModal';
import { ToastProvider, useToast } from './components/Toast';
import { api, getStoredToken, setStoredToken, BACKEND_BASE_URL } from './services/api';
import { Radio, Database, Server, Cpu, CheckCircle2, Heart } from 'lucide-react';

function MainApp() {
  const { showToast } = useToast();
  const [health, setHealth] = useState<HealthCheckResponse | null>(null);
  const [selectedPollId, setSelectedPollId] = useState<string | null>(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
  const [isHealthModalOpen, setIsHealthModalOpen] = useState(false);
  const [pollToDelete, setPollToDelete] = useState<Poll | null>(null);

  // Authentication State
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authToken, setAuthToken] = useState<string | null>(null);

  const workspaceRef = useRef<HTMLDivElement | null>(null);

  // Initialize auth from localStorage & /api/auth/me
  useEffect(() => {
    const token = getStoredToken();
    const savedUser = localStorage.getItem('live_poll_user');

    if (token) {
      setAuthToken(token);
      if (savedUser) {
        try {
          setCurrentUser(JSON.parse(savedUser));
        } catch {
          // ignore
        }
      }

      // Verify token validity against /api/auth/me
      api
        .getCurrentUser()
        .then((user) => {
          setCurrentUser(user);
          localStorage.setItem('live_poll_user', JSON.stringify(user));
        })
        .catch(() => {
          // Token expired or invalid
          setStoredToken(null);
          localStorage.removeItem('live_poll_user');
          setAuthToken(null);
          setCurrentUser(null);
        });
    }

    // Check URL hash for initial poll selection e.g. #poll-id
    const hash = window.location.hash.replace('#', '');
    if (hash && hash.length === 24) {
      setSelectedPollId(hash);
    }
  }, []);

  // Update hash when poll changes
  const handleSelectPoll = (pollId: string) => {
    setSelectedPollId(pollId);
    window.location.hash = pollId;
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const handleBackToPolls = () => {
    setSelectedPollId(null);
    window.location.hash = '';
  };

  const handleAuthSuccess = (token: string, user: User) => {
    setAuthToken(token);
    setCurrentUser(user);
  };

  const handleLogout = () => {
    setStoredToken(null);
    localStorage.removeItem('live_poll_user');
    setAuthToken(null);
    setCurrentUser(null);
    showToast('You have signed out successfully.', 'info', 'Signed Out');
  };

  // Health check query
  const checkHealth = useCallback(async () => {
    try {
      const res = await fetch(`${BACKEND_BASE_URL}/api/health`);
      const data = (await res.json().catch(() => null)) as HealthCheckResponse | null;
      if (res.ok && data) {
        setHealth(data);
      }
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    checkHealth();
    const timer = setInterval(checkHealth, 15000);
    return () => clearInterval(timer);
  }, [checkHealth]);

  const scrollToWorkspace = () => {
    workspaceRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  return (
    <div className="min-h-screen bg-[#070a11] text-slate-100 flex flex-col font-sans selection:bg-sky-500/30 selection:text-sky-200 antialiased">
      {/* Primary Sticky Header */}
      <Navigation
        currentUser={currentUser}
        health={health}
        onOpenCreatePoll={() => {
          if (!authToken) {
            setIsAuthModalOpen(true);
          } else {
            setIsCreateModalOpen(true);
          }
        }}
        onOpenAuth={() => setIsAuthModalOpen(true)}
        onLogout={handleLogout}
        onOpenHealth={() => setIsHealthModalOpen(true)}
        onHomeClick={handleBackToPolls}
      />

      {/* Main Content View */}
      <main className="flex-1 w-full max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
        {selectedPollId ? (
          /* Live Single Poll View (Voting & Realtime Results) */
          <LivePollView
            pollId={selectedPollId}
            currentUser={currentUser}
            authToken={authToken}
            onBack={handleBackToPolls}
            onRequireAuth={() => setIsAuthModalOpen(true)}
            onPollDeleted={() => {
              handleBackToPolls();
            }}
          />
        ) : (
          /* Landing & Polls Workspace View */
          <div className="space-y-10">
            {/* Editorial Hero with Live Demo Card */}
            <HeroSection
              onCreatePollClick={() => {
                if (!authToken) {
                  setIsAuthModalOpen(true);
                } else {
                  setIsCreateModalOpen(true);
                }
              }}
              onExploreClick={scrollToWorkspace}
            />

            {/* Polls Workspace Directory */}
            <div ref={workspaceRef} className="pt-4">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h2 className="text-xl font-bold text-white tracking-tight">
                    Live Polling Workspace
                  </h2>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Browse active polls, inspect live vote distributions, and create new questions.
                  </p>
                </div>
              </div>

              <PollList
                currentUser={currentUser}
                onSelectPoll={handleSelectPoll}
                onCreatePollClick={() => {
                  if (!authToken) {
                    setIsAuthModalOpen(true);
                  } else {
                    setIsCreateModalOpen(true);
                  }
                }}
                onOpenDeleteModal={(poll) => setPollToDelete(poll)}
              />
            </div>
          </div>
        )}
      </main>

      {/* Observability & Health Modal */}
      <HealthModal
        isOpen={isHealthModalOpen}
        onClose={() => setIsHealthModalOpen(false)}
        health={health}
      />

      {/* Create Poll Modal with 2-Column Live Preview */}
      <CreatePollModal
        isOpen={isCreateModalOpen}
        authToken={authToken}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={(id) => handleSelectPoll(id)}
        onRequireAuth={() => {
          setIsCreateModalOpen(false);
          setIsAuthModalOpen(true);
        }}
      />

      {/* Authentication Modal */}
      <AuthModal
        isOpen={isAuthModalOpen}
        onClose={() => setIsAuthModalOpen(false)}
        onAuthSuccess={handleAuthSuccess}
      />

      {/* Delete Poll Modal from List View */}
      {pollToDelete && (
        <DeletePollModal
          isOpen={!!pollToDelete}
          pollId={pollToDelete.id}
          pollTitle={pollToDelete.title}
          onClose={() => setPollToDelete(null)}
          onDeleted={() => {
            showToast(`Poll "${pollToDelete.title}" deleted.`, 'info', 'Poll Deleted');
            setPollToDelete(null);
            if (selectedPollId === pollToDelete.id) {
              handleBackToPolls();
            }
          }}
          onError={(msg) => {
            showToast(msg, 'error', 'Delete Failed');
          }}
        />
      )}

      {/* Production Technical Footer */}
      <footer className="mt-auto border-t border-slate-800/70 bg-slate-950/60 py-8 px-4 sm:px-6 lg:px-8">
        <div className="max-w-6xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-slate-400">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-lg bg-sky-500/10 border border-sky-500/20 flex items-center justify-center text-sky-400">
              <Radio className="w-3.5 h-3.5" />
            </div>
            <span className="font-semibold text-slate-300">PULSE Live Polling Platform</span>
            <span className="text-slate-600">·</span>
            <span className="font-mono text-slate-500">Go 1.22 · MongoDB 7 · Redis Pub/Sub</span>
          </div>

          <div className="flex items-center gap-4 text-[11px] text-slate-500 font-mono">
            <span>Transport: Server-Sent Events (SSE)</span>
            <button
              type="button"
              onClick={() => setIsHealthModalOpen(true)}
              className="text-sky-400 hover:text-sky-300 cursor-pointer font-sans text-xs underline underline-offset-2"
            >
              System Health
            </button>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default function App() {
  return (
    <ToastProvider>
      <MainApp />
    </ToastProvider>
  );
}
