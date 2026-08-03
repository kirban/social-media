import { request } from './client'
import type { DialogMessage, DialogSummary, Post, PostFeed, User } from './types'

export interface RegisterInput {
  first_name: string
  second_name: string
  password: string
  birthdate?: string
  biography?: string
  city?: string
}

export const authApi = {
  login: (id: string, password: string) =>
    request<{ token: string }>('/login', {
      method: 'POST',
      body: { id, password },
      anonymous: true,
    }),

  register: (input: RegisterInput) =>
    request<{ user_id: string }>('/user/register', {
      method: 'POST',
      body: input,
      anonymous: true,
    }),
}

export const usersApi = {
  getById: (id: string) => request<User>(`/user/get/${id}`),

  /**
   * Both name fragments are required by the API — it returns 400 if either is
   * missing — so callers must not fire this with a single field filled in.
   */
  search: (firstName: string, lastName: string) =>
    request<User[]>('/user/search', { query: { first_name: firstName, last_name: lastName } }),
}

export const friendsApi = {
  list: () => request<User[]>('/friend/list'),
  add: (userId: string) => request<void>(`/friend/set/${userId}`, { method: 'PUT' }),
  remove: (userId: string) => request<void>(`/friend/delete/${userId}`, { method: 'PUT' }),
}

export const postsApi = {
  feed: (limit: number, offset: number) =>
    request<PostFeed>('/post/feed', { query: { limit, offset } }),

  byUser: (userId: string, limit: number, offset: number) =>
    request<PostFeed>(`/post/list/${userId}`, { query: { limit, offset } }),

  getById: (id: string) => request<Post>(`/post/get/${id}`),

  create: (text: string) =>
    request<{ post_id: string }>('/post/create', { method: 'POST', body: { text } }),

  update: (id: string, text: string) =>
    request<void>('/post/update', { method: 'PUT', body: { id, text } }),

  // Deletion is a PUT, not a DELETE — that is what the API defines.
  remove: (id: string) => request<void>(`/post/delete/${id}`, { method: 'PUT' }),
}

export const dialogsApi = {
  list: () => request<DialogSummary[]>('/dialog/list'),

  messages: (userId: string, limit: number, offset: number) =>
    request<{ messages: DialogMessage[]; limit: number; offset: number }>(
      `/dialog/${userId}/list`,
      { query: { limit, offset } },
    ),

  send: (userId: string, text: string) =>
    request<{ id: string }>(`/dialog/${userId}/send`, { method: 'POST', body: { text } }),
}
