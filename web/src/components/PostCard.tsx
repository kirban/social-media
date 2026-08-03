import { Card, Space, Typography, Button, Popconfirm, Input, Avatar } from 'antd'
import { EditOutlined, DeleteOutlined, UserOutlined } from '@ant-design/icons'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import dayjs from 'dayjs'
import type { Post, User } from '../api/types'
import { fullName } from '../api/types'

interface PostCardProps {
  post: Post
  /** Author details, when already loaded. Falls back to the raw id. */
  author?: User
  editable?: boolean
  onUpdate?: (id: string, text: string) => void
  onDelete?: (id: string) => void
  busy?: boolean
}

export function PostCard({ post, author, editable, onUpdate, onDelete, busy }: PostCardProps) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(post.text ?? '')

  const authorId = post.created_by ?? ''
  const authorLabel = author ? fullName(author) : authorId

  const save = () => {
    const text = draft.trim()
    if (!text || !post.id) return
    onUpdate?.(post.id, text)
    setEditing(false)
  }

  return (
    <Card
      size="small"
      title={
        <Space>
          <Avatar size="small" icon={<UserOutlined />} />
          {authorId ? <Link to={`/users/${authorId}`}>{authorLabel}</Link> : authorLabel}
        </Space>
      }
      extra={
        post.created_at ? (
          <Typography.Text type="secondary">
            {dayjs(post.created_at).format('D MMM YYYY HH:mm')}
          </Typography.Text>
        ) : null
      }
      actions={
        editable
          ? [
              <Button key="edit" type="text" icon={<EditOutlined />} onClick={() => setEditing(true)}>
                Edit
              </Button>,
              <Popconfirm
                key="delete"
                title="Delete this post?"
                okText="Delete"
                okButtonProps={{ danger: true }}
                onConfirm={() => post.id && onDelete?.(post.id)}
              >
                <Button type="text" danger icon={<DeleteOutlined />} loading={busy}>
                  Delete
                </Button>
              </Popconfirm>,
            ]
          : undefined
      }
    >
      {editing ? (
        <Space orientation="vertical" style={{ width: '100%' }}>
          <Input.TextArea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            autoSize={{ minRows: 2, maxRows: 8 }}
          />
          <Space>
            <Button type="primary" onClick={save} loading={busy}>
              Save
            </Button>
            <Button
              onClick={() => {
                setDraft(post.text ?? '')
                setEditing(false)
              }}
            >
              Cancel
            </Button>
          </Space>
        </Space>
      ) : (
        <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>
          {post.text}
        </Typography.Paragraph>
      )}
    </Card>
  )
}
