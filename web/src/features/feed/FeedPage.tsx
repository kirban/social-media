import { useState } from 'react'
import { Card, Input, Button, Space, Typography, Empty, Alert, Skeleton, App } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { postsApi, friendsApi } from '../../api/resources'
import { queryKeys } from '../../api/queryKeys'
import { PostCard } from '../../components/PostCard'
import type { User } from '../../api/types'

const PAGE_SIZE = 10

export function FeedPage() {
  const [page, setPage] = useState(1)
  const [draft, setDraft] = useState('')
  const queryClient = useQueryClient()
  const { message } = App.useApp()

  const feed = useQuery({
    queryKey: [...queryKeys.feed, page],
    queryFn: () => postsApi.feed(PAGE_SIZE, (page - 1) * PAGE_SIZE),
  })

  // Friends are already fetched for the Friends screen; reusing the query here
  // turns author ids into names without an extra request per post.
  const friends = useQuery({ queryKey: queryKeys.friends, queryFn: friendsApi.list })
  const authorsById = new Map<string, User>(
    (friends.data ?? []).filter((u): u is User & { id: string } => Boolean(u.id)).map((u) => [u.id, u]),
  )

  const createPost = useMutation({
    mutationFn: (text: string) => postsApi.create(text),
    onSuccess: () => {
      setDraft('')
      message.success('Posted')
      // Your own posts do not come back over the WebSocket — the fan-out targets
      // your followers, not you — so the feed is refreshed explicitly here.
      void queryClient.invalidateQueries({ queryKey: queryKeys.feed })
    },
    onError: (error: Error) => message.error(error.message),
  })

  const posts = feed.data?.posts ?? []
  // The API returns no total count, so there is no honest way to render numbered
  // pages. A full page implies there may be another one.
  const hasNextPage = posts.length === PAGE_SIZE

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Card size="small">
        <Space orientation="vertical" style={{ width: '100%' }}>
          <Input.TextArea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="What's on your mind?"
            autoSize={{ minRows: 2, maxRows: 6 }}
            maxLength={2000}
            showCount
          />
          <Button
            type="primary"
            disabled={!draft.trim()}
            loading={createPost.isPending}
            onClick={() => createPost.mutate(draft.trim())}
          >
            Post
          </Button>
        </Space>
      </Card>

      {feed.isError && <Alert type="error" showIcon title={(feed.error as Error).message} />}

      {feed.isPending ? (
        <Card>
          <Skeleton active paragraph={{ rows: 3 }} />
        </Card>
      ) : posts.length === 0 ? (
        <Card>
          <Empty
            description={
              <Space orientation="vertical">
                <Typography.Text>Your feed is empty.</Typography.Text>
                <Typography.Text type="secondary">
                  Your feed shows posts from people you follow — find some on the Search page.
                </Typography.Text>
              </Space>
            }
          />
        </Card>
      ) : (
        <>
          {posts.map((post) => (
            <PostCard
              key={post.id}
              post={post}
              author={post.created_by ? authorsById.get(post.created_by) : undefined}
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
