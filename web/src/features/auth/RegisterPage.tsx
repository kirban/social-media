import { Card, Form, Input, Button, Typography, Alert, Space, DatePicker } from 'antd'
import { useMutation } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import type { Dayjs } from 'dayjs'
import { authApi } from '../../api/resources'

interface RegisterForm {
  first_name: string
  second_name: string
  password: string
  birthdate?: Dayjs
  biography?: string
  city?: string
}

export function RegisterPage() {
  const navigate = useNavigate()

  const mutation = useMutation({
    mutationFn: (values: RegisterForm) =>
      authApi.register({
        first_name: values.first_name.trim(),
        second_name: values.second_name.trim(),
        password: values.password,
        // The API expects a plain date, not a timestamp.
        birthdate: values.birthdate?.format('YYYY-MM-DD'),
        biography: values.biography?.trim(),
        city: values.city?.trim(),
      }),
    onSuccess: (data) => {
      // There is no auto-login endpoint, so hand the new id to the sign-in
      // screen rather than making the user copy it from a toast.
      navigate('/login', { state: { registeredId: data.user_id } })
    },
  })

  return (
    <div style={{ display: 'grid', placeItems: 'center', minHeight: '100vh', padding: 24 }}>
      <Card style={{ width: '100%', maxWidth: 460 }}>
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Title level={3} style={{ marginBottom: 0 }}>
            Create an account
          </Typography.Title>

          {mutation.isError && (
            <Alert type="error" showIcon title={(mutation.error as Error).message} />
          )}

          <Form<RegisterForm> layout="vertical" onFinish={(values) => mutation.mutate(values)}>
            <Form.Item
              name="first_name"
              label="First name"
              rules={[{ required: true, message: 'Enter your first name' }]}
            >
              <Input />
            </Form.Item>

            <Form.Item
              name="second_name"
              label="Last name"
              rules={[{ required: true, message: 'Enter your last name' }]}
            >
              <Input />
            </Form.Item>

            <Form.Item
              name="password"
              label="Password"
              rules={[{ required: true, message: 'Choose a password' }]}
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>

            <Form.Item name="birthdate" label="Date of birth">
              <DatePicker style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item name="city" label="City">
              <Input />
            </Form.Item>

            <Form.Item name="biography" label="About you">
              <Input.TextArea autoSize={{ minRows: 2, maxRows: 5 }} />
            </Form.Item>

            <Button type="primary" htmlType="submit" block loading={mutation.isPending}>
              Register
            </Button>
          </Form>

          <Typography.Text type="secondary">
            Already registered? <Link to="/login">Sign in</Link>
          </Typography.Text>
        </Space>
      </Card>
    </div>
  )
}
