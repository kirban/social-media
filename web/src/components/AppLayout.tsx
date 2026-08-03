import { Layout, Menu, Typography, Button, Space, notification } from 'antd'
import {
  TeamOutlined,
  SearchOutlined,
  MessageOutlined,
  UserOutlined,
  ProfileOutlined,
  LogoutOutlined,
} from '@ant-design/icons'
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useCallback } from 'react'
import { useAuthStore } from '../auth/store'
import { useFeedSocket } from '../ws/useFeedSocket'
import type { FeedPostedMessage } from '../api/types'

const { Header, Content } = Layout

const NAV_ITEMS = [
  { key: '/feed', icon: <ProfileOutlined />, label: <Link to="/feed">Feed</Link> },
  { key: '/friends', icon: <TeamOutlined />, label: <Link to="/friends">Friends</Link> },
  { key: '/search', icon: <SearchOutlined />, label: <Link to="/search">Search</Link> },
  { key: '/dialogs', icon: <MessageOutlined />, label: <Link to="/dialogs">Messages</Link> },
  { key: '/me', icon: <UserOutlined />, label: <Link to="/me">Profile</Link> },
]

export function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const signOut = useAuthStore((state) => state.signOut)
  const [api, contextHolder] = notification.useNotification()

  const announcePost = useCallback(
    (message: FeedPostedMessage) => {
      api.info({
        title: 'New post in your feed',
        description: message.postText.slice(0, 120),
        placement: 'bottomRight',
        duration: 4,
      })
    },
    [api],
  )

  // Mounted once at the layout level so a single socket serves every screen.
  useFeedSocket(announcePost)

  // Highlight the nav entry whose route prefixes the current path, so
  // /dialogs/<id> still marks Messages as active.
  const selected = NAV_ITEMS.map((item) => item.key).filter((key) =>
    location.pathname.startsWith(key),
  )

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {contextHolder}
      <Header style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
        <Typography.Title level={4} style={{ color: '#fff', margin: 0, whiteSpace: 'nowrap' }}>
          Social
        </Typography.Title>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={selected}
          items={NAV_ITEMS}
          style={{ flex: 1, minWidth: 0 }}
        />
        <Space>
          <Button
            icon={<LogoutOutlined />}
            onClick={() => {
              signOut()
              navigate('/login')
            }}
          >
            Sign out
          </Button>
        </Space>
      </Header>
      <Content style={{ padding: 24 }}>
        <div style={{ maxWidth: 900, margin: '0 auto' }}>
          <Outlet />
        </div>
      </Content>
    </Layout>
  )
}
