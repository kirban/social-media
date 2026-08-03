interface TokenClaims {
  user_id: string
  /** Expiry, in seconds since the epoch. */
  exp: number
}

function decodeBase64Url(segment: string): string {
  const padded = segment.replace(/-/g, '+').replace(/_/g, '/')
  const withPadding = padded.padEnd(padded.length + ((4 - (padded.length % 4)) % 4), '=')

  // decodeURIComponent/escape round-trip so multi-byte characters survive atob.
  return decodeURIComponent(
    atob(withPadding)
      .split('')
      .map((c) => '%' + c.charCodeAt(0).toString(16).padStart(2, '0'))
      .join(''),
  )
}

/**
 * Reads the claims out of a JWT without verifying it.
 *
 * The signature is the server's business; the client only needs the user id, so
 * it can tell whose profile is "mine", and the expiry, so an obviously dead
 * token is discarded on startup rather than after a failed request.
 */
export function readClaims(token: string): TokenClaims | null {
  const parts = token.split('.')
  if (parts.length !== 3) return null

  try {
    const claims = JSON.parse(decodeBase64Url(parts[1])) as Partial<TokenClaims>
    if (typeof claims.user_id !== 'string' || typeof claims.exp !== 'number') return null
    return { user_id: claims.user_id, exp: claims.exp }
  } catch {
    return null
  }
}

export function isExpired(claims: TokenClaims): boolean {
  return claims.exp * 1000 <= Date.now()
}
