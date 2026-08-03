/** Central query keys so cache invalidation stays consistent across features. */
export const queryKeys = {
  feed: ['feed'] as const,
  friends: ['friends'] as const,
  user: (id: string) => ['user', id] as const,
  userPosts: (id: string) => ['posts', 'byUser', id] as const,
  search: (firstName: string, lastName: string) => ['search', firstName, lastName] as const,
  dialogs: ['dialogs'] as const,
  dialogMessages: (userId: string) => ['dialog', userId] as const,
}
