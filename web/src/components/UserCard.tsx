import { Avatar, Button, Space, Typography, Flex, Divider, Empty, Skeleton } from 'antd'
import { UserOutlined, MessageOutlined } from '@ant-design/icons'
import { Link } from 'react-router-dom'
import type { ReactNode } from 'react'
import type { User } from '../api/types'
import { fullName } from '../api/types'

interface UserRowProps {
  user: User
  action?: ReactNode
}

/** One person in a list: avatar, name, a line of detail, and actions. */
export function UserRow({ user, action }: UserRowProps) {
  const details = [user.city, user.biography].filter(Boolean).join(' · ')

  return (
    <Flex align="center" justify="space-between" gap="middle" style={{ padding: '12px 0' }}>
      <Flex align="center" gap="middle" style={{ minWidth: 0 }}>
        <Avatar icon={<UserOutlined />} />
        <div style={{ minWidth: 0 }}>
          <div>
            {user.id ? (
              <Link to={`/users/${user.id}`}>
                <strong>{fullName(user)}</strong>
              </Link>
            ) : (
              <strong>{fullName(user)}</strong>
            )}
          </div>
          <Typography.Text type="secondary" ellipsis>
            {details || 'No details'}
          </Typography.Text>
        </div>
      </Flex>

      <Space>
        {user.id && (
          <Link to={`/dialogs/${user.id}`}>
            <Button icon={<MessageOutlined />}>Message</Button>
          </Link>
        )}
        {action}
      </Space>
    </Flex>
  )
}

interface UserListProps {
  users: User[]
  loading?: boolean
  empty?: ReactNode
  renderAction?: (user: User) => ReactNode
}

/**
 * A vertical list of people with dividers between rows.
 *
 * Hand-rolled rather than using antd's List, which is deprecated in v6 and
 * slated for removal.
 */
export function UserList({ users, loading, empty, renderAction }: UserListProps) {
  if (loading) return <Skeleton active avatar paragraph={{ rows: 2 }} />
  if (users.length === 0) return <Empty description={empty ?? 'Nothing here yet.'} />

  return (
    <div>
      {users.map((user, index) => (
        <div key={user.id ?? index}>
          {index > 0 && <Divider style={{ margin: 0 }} />}
          <UserRow user={user} action={renderAction?.(user)} />
        </div>
      ))}
    </div>
  )
}
