import { Navigate, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuthStore } from './store'

/**
 * Gates the authenticated area. A 401 anywhere clears the token, which
 * re-renders this and bounces the user to the sign-in screen with the page they
 * wanted preserved.
 */
export function ProtectedRoute({ children }: { children: ReactNode }) {
  const token = useAuthStore((state) => state.token)
  const location = useLocation()

  if (!token) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <>{children}</>
}
