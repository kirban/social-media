import { Card, Form, Input, Button, Typography, Alert, Space } from 'antd'
import { useMutation } from '@tanstack/react-query'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { authApi } from '../../api/resources'
import { ApiError } from '../../api/client'
import { useAuthStore } from '../../auth/store'

interface LoginForm {
  id: string
  password: string
}

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const signIn = useAuthStore((state) => state.signIn)

  // Set by ProtectedRoute when it bounced an unauthenticated visitor, and by
  // the register screen after it creates an account.
  const state = location.state as { from?: string; registeredId?: string } | null

  const mutation = useMutation({
    mutationFn: ({ id, password }: LoginForm) => authApi.login(id.trim(), password),
    onSuccess: (data) => {
      signIn(data.token)
      navigate(state?.from ?? '/feed', { replace: true })
    },
  })

  return (
    <div style={{ display: 'grid', placeItems: 'center', minHeight: '100vh', padding: 24 }}>
      <Card style={{ width: '100%', maxWidth: 420 }}>
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Title level={3} style={{ marginBottom: 0 }}>
            Sign in
          </Typography.Title>

          {state?.registeredId && (
            <Alert
              type="success"
              showIcon
              title="Account created"
              description="Your user ID has been filled in below. Keep it — it is how you sign in."
            />
          )}

          {mutation.isError && (
            <Alert
              type="error"
              showIcon
              title={
                mutation.error instanceof ApiError && mutation.error.status === 404
                  ? 'No user with that ID.'
                  : (mutation.error as Error).message
              }
            />
          )}

          <Form<LoginForm>
            layout="vertical"
            initialValues={{ id: state?.registeredId ?? '', password: '' }}
            onFinish={(values) => mutation.mutate(values)}
          >
            <Form.Item
              name="id"
              label="User ID"
              extra="The UUID you received when registering."
              rules={[{ required: true, message: 'Enter your user ID' }]}
            >
              <Input placeholder="e4d2e6b0-cde2-42c5-aac3-0b8316f21e58" autoComplete="username" />
            </Form.Item>

            <Form.Item
              name="password"
              label="Password"
              rules={[{ required: true, message: 'Enter your password' }]}
            >
              <Input.Password autoComplete="current-password" />
            </Form.Item>

            <Button type="primary" htmlType="submit" block loading={mutation.isPending}>
              Sign in
            </Button>
          </Form>

          <Typography.Text type="secondary">
            No account? <Link to="/register">Register</Link>
          </Typography.Text>
        </Space>
      </Card>
    </div>
  )
}
