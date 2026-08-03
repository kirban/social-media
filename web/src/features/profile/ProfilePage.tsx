import { useState } from 'react'
import {
  Card,
  Descriptions,
  Space,
  Typography,
  Alert,
  Skeleton,
  Button,
  Empty,
  App,
  Avatar,
} from 'antd'
import { UserOutlined, MessageOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import dayjs from 'dayjs'
import { postsApi, usersApi } from '../../api/resources'
import { queryKeys } from '../../api/queryKeys'
import { useAuthStore } from '../../auth/store'
import { PostCard } from '../../components/PostCard'
import { FollowButton } from '../friends/FollowButton'
import { fullName } from '../../api/types'

const PAGE_SIZE = 10

/** Renders both /me and /users/:id — the only difference is whose id is used. */
export function ProfilePage({ self = false }: { self?: boolean }) {
  const params = useParams<{ id: string }>()
  const currentUserId = useAuthStore((state) => state.userId)
  const userId = self ? currentUserId : params.id
  const isOwnProfile = Boolean(userId && userId === currentUserId)

  const [page, setPage] = useState(1)
  const queryClient = useQueryClient()
  const { message } = App.useApp()

  const user = useQuery({
    queryKey: queryKeys.user(userId ?? ''),
    queryFn: () => usersApi.getById(userId!),
    enabled: Boolean(userId),
  })

  const posts = useQuery({
    queryKey: [...queryKeys.userPosts(userId ?? ''), page],
    queryFn: () => postsApi.byUser(userId!, PAGE_SIZE, (page - 1) * PAGE_SIZE),
    enabled: Boolean(userId),
  })

  const invalidatePosts = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.userPosts(userId ?? '') })
    void queryClient.invalidateQueries({ queryKey: queryKeys.feed })
  }

  const updatePost = useMutation({
    mutationFn: ({ id, text }: { id: string; text: string }) => postsApi.update(id, text),
    onSuccess: () => {
      message.success('Post updated')
      invalidatePosts()
    },
    onError: (error: Error) => message.error(error.message),
  })

  const deletePost = useMutation({
    mutationFn: (id: string) => postsApi.remove(id),
    onSuccess: () => {
      message.success('Post deleted')
      invalidatePosts()
    },
    onError: (error: Error) => message.error(error.message),
  })

  if (!userId) {
    return <Alert type="error" showIcon title="No user selected." />
  }

  if (user.isError) {
    return <Alert type="error" showIcon title={(user.error as Error).message} />
  }

  const items = posts.data?.posts ?? []
  const hasNextPage = items.length === PAGE_SIZE

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Card>
        {user.isPending || !user.data ? (
          <Skeleton active avatar paragraph={{ rows: 3 }} />
        ) : (
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <Space align="center" size="middle">
              <Avatar size={56} icon={<UserOutlined />} />
              <Typography.Title level={4} style={{ margin: 0 }}>
                {fullName(user.data)}
              </Typography.Title>
            </Space>

            <Descriptions
              column={1}
              size="small"
              items={[
                { key: 'city', label: 'City', children: user.data.city || '—' },
                {
                  key: 'born',
                  label: 'Born',
                  children: user.data.birthdate
                    ? dayjs(user.data.birthdate).format('D MMMM YYYY')
                    : '—',
                },
                { key: 'about', label: 'About', children: user.data.biography || '—' },
                {
                  key: 'id',
                  label: 'User ID',
                  children: (
                    <Typography.Text copyable code>
                      {userId}
                    </Typography.Text>
                  ),
                },
              ]}
            />

            {!isOwnProfile && (
              <Space>
                <FollowButton userId={userId} />
                <Link to={`/dialogs/${userId}`}>
                  <Button icon={<MessageOutlined />}>Message</Button>
                </Link>
              </Space>
            )}
          </Space>
        )}
      </Card>

      <Typography.Title level={5} style={{ marginBottom: 0 }}>
        {isOwnProfile ? 'Your posts' : 'Posts'}
      </Typography.Title>

      {posts.isError && <Alert type="error" showIcon title={(posts.error as Error).message} />}

      {posts.isPending ? (
        <Card>
          <Skeleton active paragraph={{ rows: 2 }} />
        </Card>
      ) : items.length === 0 ? (
        <Card>
          <Empty description={isOwnProfile ? 'You have not posted yet.' : 'No posts yet.'} />
        </Card>
      ) : (
        <>
          {items.map((post) => (
            <PostCard
              key={post.id}
              post={post}
              author={user.data}
              editable={isOwnProfile}
              busy={updatePost.isPending || deletePost.isPending}
              onUpdate={(id, text) => updatePost.mutate({ id, text })}
              onDelete={(id) => deletePost.mutate(id)}
            />
          ))}
          <Space style={{ justifyContent: 'center', width: '100%' }}>
            <Button disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
              Previous
            </Button>
            <Typography.Text type="secondary">Page {page}</Typography.Text>
            <Button disabled={!hasNextPage} onClick={() => setPage((p) => p + 1)}>
              Next
            </Button>
          </Space>
        </>
      )}
    </Space>
  )
}
