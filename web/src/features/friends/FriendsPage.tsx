import { Card, Alert, Typography, Space } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { friendsApi } from '../../api/resources'
import { queryKeys } from '../../api/queryKeys'
import { UserList } from '../../components/UserCard'
import { FollowButton } from './FollowButton'

export function FriendsPage() {
  const friends = useQuery({ queryKey: queryKeys.friends, queryFn: friendsApi.list })

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Typography.Title level={4} style={{ marginBottom: 0 }}>
        Friends
      </Typography.Title>
      <Typography.Text type="secondary">
        People you follow. Their posts appear in your feed.
      </Typography.Text>

      {friends.isError && <Alert type="error" showIcon title={(friends.error as Error).message} />}

      <Card>
        <UserList
          users={friends.data ?? []}
          loading={friends.isPending}
          empty={
            <span>
              You are not following anyone yet. <Link to="/search">Find people</Link>.
            </span>
          }
          renderAction={(user) => (user.id ? <FollowButton userId={user.id} /> : null)}
        />
      </Card>
    </Space>
  )
}
