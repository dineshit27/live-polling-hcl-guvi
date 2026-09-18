// Utility to retrieve or generate a unique persistent client fingerprint for guest voting
export function getVoterIdentifier(): string {
  const STORAGE_KEY = 'live_poll_voter_id';
  let voterId = localStorage.getItem(STORAGE_KEY);
  if (!voterId) {
    voterId = 'guest_' + Math.random().toString(36).substring(2, 11) + '_' + Date.now().toString(36);
    try {
      localStorage.setItem(STORAGE_KEY, voterId);
    } catch {
      // Ignore if localStorage is restricted
    }
  }
  return voterId;
}
