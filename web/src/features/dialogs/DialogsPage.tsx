import { Card, Empty, Alert, Typography, Space, Avatar, Skeleton, Flex, Divider } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import { useQueries, useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { dialogsApi, usersApi } from '../../api/resources'
import { queryKeys } from '../../api/queryKeys'
import { fullName } from '../../api/types'

/**
 * Conversation list.
 *
 * The dialog endpoint returns only peer ids — carrying a last-message preview
 * would force a cross-shard query server-side — so each peer's name is resolved
 * with a separate profile lookup. These are cached per user and shared with the
 * profile screens.
 */
export function DialogsPage() {
  const dialogs = useQuery({ queryKey: queryKeys.dialogs, queryFn: dialogsApi.list })

  const peerIds = (dialogs.data ?? []).map((d) => d.user_id).filter((id): id is string => Boolean(id))

  const peers = useQueries({
    queries: peerIds.map((id) => ({
      queryKey: queryKeys.user(id),
      queryFn: () => usersApi.getById(id),
    })),
  })

  const nameFor = (peerId: string) => {
    const index = peerIds.indexOf(peerId)
    const peer = index >= 0 ? peers[index] : undefined
    if (peer?.isPending) return null
    return peer?.data ? fullName(peer.data) : peerId
  }

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Typography.Title level={4} style={{ marginBottom: 0 }}>
        Messages
      </Typography.Title>

      {dialogs.isError && <Alert type="error" showIcon title={(dialogs.error as Error).message} />}

      <Card>
        {dialogs.isPending ? (
          <Skeleton active avatar paragraph={{ rows: 2 }} />
        ) : (dialogs.data ?? []).length === 0 ? (
          <Empty
            description={
              <span>
                No conversations yet. Open someone's profile from <Link to="/search">Search</Link> and
                send them a message.
              </span>
            }
          />
        ) : (
          (dialogs.data ?? []).map((dialog, index) => {
            const name = nameFor(dialog.user_id)
            return (
              <div key={dialog.dialog_id}>
                {index > 0 && <Divider style={{ margin: 0 }} />}
                <Link to={`/dialogs/${dialog.user_id}`}>
                  <Flex align="center" gap="middle" style={{ padding: '12px 0' }}>
                    <Avatar icon={<UserOutlined />} />
                    {name ?? <Skeleton.Input active size="small" style={{ width: 140 }} />}
                  </Flex>
                </Link>
              </div>
            )
          })
        )}
      </Card>
    </Space>
  )
}
