import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAuthStore } from '../auth/store'
import { queryKeys } from '../api/queryKeys'
import type { FeedPostedMessage } from '../api/types'

const WS_PATH = '/api/v1/post/feed/posted'

const INITIAL_RETRY_MS = 1000
const MAX_RETRY_MS = 30_000

function socketUrl(token: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  // The token travels as a query parameter because the browser WebSocket
  // constructor cannot set an Authorization header. The backend accepts this
  // only on upgrade requests.
  return `${protocol}//${window.location.host}${WS_PATH}?token=${encodeURIComponent(token)}`
}

/**
 * Keeps a WebSocket open for as long as the user is signed in, invalidating the
 * feed whenever a followed account posts.
 *
 * The server pushes only post-created events, and drops clients whose send
 * buffer backs up, so reconnecting must be routine rather than exceptional.
 */
export function useFeedSocket(onPost?: (message: FeedPostedMessage) => void): void {
  const token = useAuthStore((state) => state.token)
  const queryClient = useQueryClient()

  // Held in a ref so reconnects always call the latest callback without
  // tearing down the socket every time the component re-renders.
  const onPostRef = useRef(onPost)
  useEffect(() => {
    onPostRef.current = onPost
  }, [onPost])

  useEffect(() => {
    if (!token) return

    let socket: WebSocket | null = null
    let retryDelay = INITIAL_RETRY_MS
    let retryTimer: number | undefined
    let closedByUs = false

    const connect = () => {
      socket = new WebSocket(socketUrl(token))

      socket.onopen = () => {
        retryDelay = INITIAL_RETRY_MS
      }

      socket.onmessage = (event) => {
        let message: FeedPostedMessage
        try {
          message = JSON.parse(event.data as string) as FeedPostedMessage
        } catch {
          return // ignore anything that isn't the one shape we expect
        }

        void queryClient.invalidateQueries({ queryKey: queryKeys.feed })
        void queryClient.invalidateQueries({
          queryKey: queryKeys.userPosts(message.author_user_id),
        })
        onPostRef.current?.(message)
      }

      socket.onclose = () => {
        if (closedByUs) return

        // A rejected handshake (expired token) closes immediately and would
        // otherwise spin; the backoff caps the retry rate either way.
        retryTimer = window.setTimeout(connect, retryDelay)
        retryDelay = Math.min(retryDelay * 2, MAX_RETRY_MS)
      }
    }

    connect()

    return () => {
      closedByUs = true
      window.clearTimeout(retryTimer)
      socket?.close()
    }
  }, [token, queryClient])
}
