import { useEffect } from 'react';

const DEFAULT_TITLE = 'PULSE Live — Real-Time Polling Platform';
const DEFAULT_DESCRIPTION =
  'Create, share, and vote on live polls with real-time results powered by Go, Redis Pub/Sub, MongoDB, and Server-Sent Events.';
const DEFAULT_CANONICAL = 'https://pulse-live-hclguvi.onrender.com/';

interface SEOOptions {
  title?: string;
  description?: string;
  canonical?: string | null;
  /**
   * Pass 'noindex, nofollow' for private/auth/dashboard views.
   * Defaults to 'index, follow'.
   */
  robots?: 'index, follow' | 'noindex, nofollow';
  /**
   * Whether this SEO configuration is active.
   * Defaults to true.
   */
  enabled?: boolean;
}

/**
 * Safely sanitizes user-provided string for meta/title attributes.
 * Strips HTML tags, newlines, and excessive whitespace.
 */
export function sanitizeMetaText(input?: string): string {
  if (!input) return '';
  return input
    .replace(/<[^>]*>/g, '') // remove HTML tags
    .replace(/[\r\n\t]+/g, ' ') // collapse newlines/tabs
    .replace(/\s{2,}/g, ' ') // collapse multi-spaces
    .trim();
}

/**
 * Lightweight SEO hook — sets document.title and updates meta/canonical tags in-place.
 * No external library required. Safe for React SPA environments.
 */
export function useSEO({
  title = DEFAULT_TITLE,
  description = DEFAULT_DESCRIPTION,
  canonical = DEFAULT_CANONICAL,
  robots = 'index, follow',
  enabled = true,
}: SEOOptions = {}) {
  useEffect(() => {
    if (!enabled) return;

    const cleanTitle = sanitizeMetaText(title) || DEFAULT_TITLE;
    const cleanDescription = sanitizeMetaText(description) || DEFAULT_DESCRIPTION;

    // 1. Title
    document.title = cleanTitle;

    // Helper to update or create meta tag
    const updateMeta = (selector: string, attrName: string, attrValue: string, content: string) => {
      let el = document.querySelector<HTMLMetaElement>(selector);
      if (!el) {
        el = document.createElement('meta');
        el.setAttribute(attrName, attrValue);
        document.head.appendChild(el);
      }
      el.setAttribute('content', content);
    };

    // 2. Standard Meta Description & Robots
    updateMeta('meta[name="description"]', 'name', 'description', cleanDescription);
    updateMeta('meta[name="robots"]', 'name', 'robots', robots);

    // 3. Open Graph
    updateMeta('meta[property="og:title"]', 'property', 'og:title', cleanTitle);
    updateMeta('meta[property="og:description"]', 'property', 'og:description', cleanDescription);

    // 4. Twitter Card
    updateMeta('meta[name="twitter:title"]', 'name', 'twitter:title', cleanTitle);
    updateMeta('meta[name="twitter:description"]', 'name', 'twitter:description', cleanDescription);

    // 5. Canonical Link
    let canonicalEl = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');
    if (canonical) {
      if (!canonicalEl) {
        canonicalEl = document.createElement('link');
        canonicalEl.setAttribute('rel', 'canonical');
        document.head.appendChild(canonicalEl);
      }
      canonicalEl.setAttribute('href', canonical);
    } else if (canonicalEl) {
      canonicalEl.remove();
    }
  }, [title, description, canonical, robots, enabled]);
}

export { DEFAULT_TITLE, DEFAULT_DESCRIPTION, DEFAULT_CANONICAL };
