import { Injectable, computed, signal } from '@angular/core';

/** What the API put in the session token at the end of the LTI launch. */
export interface Session {
  course_id: number;
  sub: string;
  name: string;
  instructor: boolean;
  exp: number;
}

const STORAGE_KEY = 'flashcards.token';

/**
 * Holds the session token. /lti/launch redirects to /#token=..., so on start
 * we read the token from the URL fragment, keep it in a signal, and remove it
 * from the address bar. A copy goes in sessionStorage so a page reload in the
 * same tab keeps working.
 */
@Injectable({ providedIn: 'root' })
export class SessionService {
  readonly token = signal<string | null>(null);

  /** The token's payload. Display only: the API checks the signature and role. */
  readonly session = computed<Session | null>(() => {
    const token = this.token();
    return token ? decodePayload(token) : null;
  });

  readonly isInstructor = computed(() => this.session()?.instructor ?? false);

  /** Set when the API rejects the token, so the UI can say "launch again". */
  readonly expired = signal(false);

  constructor() {
    const fromUrl = new URLSearchParams(location.hash.slice(1)).get('token');
    if (fromUrl) {
      history.replaceState(null, '', location.pathname + location.search);
      this.save(fromUrl);
    } else {
      this.token.set(readStorage());
    }
    const s = this.session();
    if (s && s.exp * 1000 <= Date.now()) {
      this.clear();
    }
  }

  clear(): void {
    this.token.set(null);
    this.expired.set(true);
    writeStorage(null);
  }

  private save(token: string): void {
    this.token.set(token);
    writeStorage(token);
  }
}

function decodePayload(token: string): Session | null {
  try {
    const part = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
    const json = new TextDecoder().decode(Uint8Array.from(atob(part), (c) => c.charCodeAt(0)));
    return JSON.parse(json) as Session;
  } catch {
    return null;
  }
}

// Storage can throw inside an LMS iframe when the browser blocks it.
function readStorage(): string | null {
  try {
    return sessionStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

function writeStorage(token: string | null): void {
  try {
    if (token) sessionStorage.setItem(STORAGE_KEY, token);
    else sessionStorage.removeItem(STORAGE_KEY);
  } catch {
    // The signal still holds the token for this page view.
  }
}
