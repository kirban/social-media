import { create } from 'zustand'
import { configureClient } from '../api/client'
import { isExpired, readClaims } from './token'

const STORAGE_KEY = 'social-media.token'

interface AuthState {
  token: string | null
  userId: string | null
  signIn: (token: string) => void
  signOut: () => void
}

/**
 * Restores a token from a previous session, discarding it if it is malformed or
 * already expired. Tokens last 24h and there is no refresh endpoint, so an
 * expired one is simply dead weight that would cause a 401 on first use.
 */
function restore(): { token: string | null; userId: string | null } {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (!stored) return { token: null, userId: null }

  const claims = readClaims(stored)
  if (!claims || isExpired(claims)) {
    localStorage.removeItem(STORAGE_KEY)
    return { token: null, userId: null }
  }

  return { token: stored, userId: claims.user_id }
}

export const useAuthStore = create<AuthState>((set) => ({
  ...restore(),

  signIn: (token: string) => {
    const claims = readClaims(token)
    if (!claims) return

    localStorage.setItem(STORAGE_KEY, token)
    set({ token, userId: claims.user_id })
  },

  signOut: () => {
    localStorage.removeItem(STORAGE_KEY)
    set({ token: null, userId: null })
  },
}))

// The client reads the token per request rather than capturing it, so a sign-in
// or sign-out takes effect immediately without re-registering.
configureClient({
  getToken: () => useAuthStore.getState().token,
  onUnauthorized: () => useAuthStore.getState().signOut(),
})
