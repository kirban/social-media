import { Button, App } from 'antd'
import { UserAddOutlined, UserDeleteOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { friendsApi } from '../../api/resources'
import { queryKeys } from '../../api/queryKeys'
import { useAuthStore } from '../../auth/store'

/**
 * Follow/unfollow toggle. Whether you already follow someone is derived from the
 * friends list rather than a per-user endpoint, since the API has no such check.
 */
export function FollowButton({ userId }: { userId: string }) {
  const queryClient = useQueryClient()
  const { message } = App.useApp()
  const currentUserId = useAuthStore((state) => state.userId)

  const friends = useQuery({ queryKey: queryKeys.friends, queryFn: friendsApi.list })
  const isFriend = (friends.data ?? []).some((user) => user.id === userId)

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.friends })
    // Following changes which posts belong in the feed.
    void queryClient.invalidateQueries({ queryKey: queryKeys.feed })
  }

  const follow = useMutation({
    mutationFn: () => friendsApi.add(userId),
    onSuccess: () => {
      message.success('Following')
      refresh()
    },
    onError: (error: Error) => message.error(error.message),
  })

  const unfollow = useMutation({
    mutationFn: () => friendsApi.remove(userId),
    onSuccess: () => {
      message.success('Unfollowed')
      refresh()
    },
    onError: (error: Error) => message.error(error.message),
  })

  // Following yourself would put your own posts in your feed and is not a
  // meaningful action.
  if (userId === currentUserId) return null

  return isFriend ? (
    <Button icon={<UserDeleteOutlined />} loading={unfollow.isPending} onClick={() => unfollow.mutate()}>
      Unfollow
    </Button>
  ) : (
    <Button
      type="primary"
      icon={<UserAddOutlined />}
      loading={follow.isPending || friends.isPending}
      onClick={() => follow.mutate()}
    >
      Follow
    </Button>
  )
}
