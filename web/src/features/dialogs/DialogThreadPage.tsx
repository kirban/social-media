import { useEffect, useRef, useState } from 'react'
import { Card, Input, Button, Space, Typography, Alert, Empty, Skeleton, App, Avatar } from 'antd'
import { SendOutlined, UserOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import dayjs from 'dayjs'
import { dialogsApi, usersApi } from '../../api/resources'
import { queryKeys } from '../../api/queryKeys'
import { useAuthStore } from '../../auth/store'
import { fullName } from '../../api/types'

const PAGE_SIZE = 100

/**
 * A single conversation.
 *
 * There is no WebSocket channel for messages — only feed posts — so incoming
 * messages arrive by polling while the thread is open.
 */
export function DialogThreadPage() {
  const { userId } = useParams<{ userId: string }>()
  const currentUserId = useAuthStore((state) => state.userId)
  const [draft, setDraft] = useState('')
  const queryClient = useQueryClient()
  const { message: toast } = App.useApp()
  const bottomRef = useRef<HTMLDivElement>(null)

  const peer = useQuery({
    queryKey: queryKeys.user(userId ?? ''),
    queryFn: () => usersApi.getById(userId!),
    enabled: Boolean(userId),
  })

  const thread = useQuery({
    queryKey: queryKeys.dialogMessages(userId ?? ''),
    queryFn: () => dialogsApi.messages(userId!, PAGE_SIZE, 0),
    enabled: Boolean(userId),
    refetchInterval: 5000,
  })

  const messages = thread.data?.messages ?? []

  const send = useMutation({
    mutationFn: (text: string) => dialogsApi.send(userId!, text),
    onSuccess: () => {
      setDraft('')
      void queryClient.invalidateQueries({ queryKey: queryKeys.dialogMessages(userId ?? '') })
      // A first message creates the dialog, so the conversation list changes.
      void queryClient.invalidateQueries({ queryKey: queryKeys.dialogs })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length])

  if (!userId) {
    return <Alert type="error" showIcon title="No conversation selected." />
  }

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Space>
        <Link to="/dialogs">
          <Button icon={<ArrowLeftOutlined />}>Back</Button>
        </Link>
        <Avatar icon={<UserOutlined />} />
        <Typography.Title level={4} style={{ margin: 0 }}>
          {peer.data ? fullName(peer.data) : userId}
        </Typography.Title>
      </Space>

      {thread.isError && <Alert type="error" showIcon title={(thread.error as Error).message} />}

      <Card styles={{ body: { maxHeight: '55vh', overflowY: 'auto' } }}>
        {thread.isPending ? (
          <Skeleton active paragraph={{ rows: 4 }} />
        ) : messages.length === 0 ? (
          <Empty description="No messages yet. Say hello." />
        ) : (
          <Space orientation="vertical" size="small" style={{ width: '100%' }}>
            {messages.map((msg, index) => {
              const mine = msg.from === currentUserId
              return (
                <div
                  // Messages carry no id in the list response, so position plus
                  // timestamp is the only stable key available.
                  key={`${msg.created_at ?? ''}-${index}`}
                  style={{ display: 'flex', justifyContent: mine ? 'flex-end' : 'flex-start' }}
                >
                  <div
                    style={{
                      maxWidth: '75%',
                      padding: '8px 12px',
                      borderRadius: 12,
                      background: mine ? '#1677ff' : '#f0f0f0',
                      color: mine ? '#fff' : 'inherit',
                    }}
                  >
                    <div style={{ whiteSpace: 'pre-wrap' }}>{msg.text}</div>
                    {msg.created_at && (
                      <div style={{ fontSize: 11, opacity: 0.7, marginTop: 4 }}>
                        {dayjs(msg.created_at).format('D MMM HH:mm')}
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
            <div ref={bottomRef} />
          </Space>
        )}
      </Card>

      <Space.Compact style={{ width: '100%' }}>
        <Input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onPressEnter={() => draft.trim() && send.mutate(draft.trim())}
          placeholder="Write a message"
          maxLength={1000}
        />
        <Button
          type="primary"
          icon={<SendOutlined />}
          loading={send.isPending}
          disabled={!draft.trim()}
          onClick={() => send.mutate(draft.trim())}
        >
          Send
        </Button>
      </Space.Compact>
    </Space>
  )
}
