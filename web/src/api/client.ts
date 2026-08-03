export const API_BASE = '/api/v1'

/** Thrown for any non-2xx response. */
export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

type TokenReader = () => string | null
type UnauthorizedHandler = () => void

let readToken: TokenReader = () => null
let onUnauthorized: UnauthorizedHandler = () => {}

/**
 * Wires the client to the auth store. Done via registration rather than a
 * direct import so the store can depend on the client without a cycle.
 */
export function configureClient(options: {
  getToken: TokenReader
  onUnauthorized: UnauthorizedHandler
}): void {
  readToken = options.getToken
  onUnauthorized = options.onUnauthorized
}

const DEFAULT_MESSAGES: Record<number, string> = {
  400: 'The server rejected that request.',
  401: 'Your session has expired. Please sign in again.',
  404: 'Not found.',
  500: 'Something went wrong on the server.',
  503: 'The server is unavailable. Try again shortly.',
}

/**
 * Reads an error message from the response.
 *
 * Several handlers answer 400 and 401 with a bare status and no body at all, so
 * parsing must tolerate an empty or non-JSON payload rather than throwing over
 * it and masking the real status.
 */
async function errorMessage(response: Response): Promise<string> {
  const fallback = DEFAULT_MESSAGES[response.status] ?? `Request failed (${response.status})`

  const raw = await response.text().catch(() => '')
  if (!raw) return fallback

  try {
    const body = JSON.parse(raw) as { message?: string }
    return body.message?.trim() || fallback
  } catch {
    // A non-JSON body (e.g. a proxy's HTML error page) tells the user nothing
    // useful; prefer the status-based message.
    return fallback
  }
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT'
  body?: unknown
  query?: Record<string, string | number | undefined>
  /** Send without an Authorization header (login and register). */
  anonymous?: boolean
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, anonymous = false } = options

  const url = new URL(API_BASE + path, window.location.origin)
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== '') url.searchParams.set(key, String(value))
  }

  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  if (!anonymous) {
    const token = readToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  let response: Response
  try {
    response = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    // fetch only rejects on a transport failure, never on an HTTP error status.
    throw new ApiError(0, 'Cannot reach the server. Is the backend running?')
  }

  if (response.status === 401) {
    // Tokens last 24h and there is no refresh endpoint, so the only recovery is
    // signing in again.
    onUnauthorized()
    throw new ApiError(401, await errorMessage(response))
  }

  if (!response.ok) {
    throw new ApiError(response.status, await errorMessage(response))
  }

  // 200-with-empty-body is the norm for the mutating endpoints.
  const raw = await response.text()
  if (!raw) return undefined as T

  return JSON.parse(raw) as T
}
