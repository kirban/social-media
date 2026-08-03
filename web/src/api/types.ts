import type { components } from './schema'

type Schemas = components['schemas']

export type User = Schemas['User']
export type Post = Schemas['Post']
export type PostFeed = Schemas['PostFeed']
export type DialogMessage = Schemas['DialogMessage']
export type DialogSummary = Schemas['DialogSummary']

/**
 * A post-created notification pushed over the feed WebSocket.
 *
 * The mixed casing is intentional — it mirrors the Go struct's JSON tags
 * (model.FeedPostedMessage), which are inconsistent. Renaming here would only
 * hide the mismatch. There is no event-type envelope and no event for post
 * updates or deletes, so this is the only shape that ever arrives.
 */
export interface FeedPostedMessage {
  postId: string
  postText: string
  author_user_id: string
}

export function fullName(user: Pick<User, 'first_name' | 'second_name'>): string {
  return [user.first_name, user.second_name].filter(Boolean).join(' ')
}
