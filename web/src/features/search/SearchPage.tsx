import { useState } from 'react'
import { Card, Form, Input, Button, Alert, Space, Typography } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { usersApi } from '../../api/resources'
import { queryKeys } from '../../api/queryKeys'
import { UserList } from '../../components/UserCard'
import { FollowButton } from '../friends/FollowButton'

interface SearchForm {
  first_name: string
  last_name: string
}

export function SearchPage() {
  const [terms, setTerms] = useState<SearchForm | null>(null)

  const results = useQuery({
    queryKey: terms ? queryKeys.search(terms.first_name, terms.last_name) : ['search', 'idle'],
    queryFn: () => usersApi.search(terms!.first_name, terms!.last_name),
    // The endpoint requires both fields and 400s otherwise, so it is only ever
    // fired once the form has supplied them.
    enabled: terms !== null,
  })

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Card>
        <Typography.Paragraph type="secondary">
          Search matches the start of both names, and the API requires both fields.
        </Typography.Paragraph>
        <Form<SearchForm>
          layout="inline"
          onFinish={(values) =>
            setTerms({ first_name: values.first_name.trim(), last_name: values.last_name.trim() })
          }
        >
          <Form.Item
            name="first_name"
            rules={[{ required: true, message: 'First name is required' }]}
          >
            <Input placeholder="First name" />
          </Form.Item>
          <Form.Item name="last_name" rules={[{ required: true, message: 'Last name is required' }]}>
            <Input placeholder="Last name" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={results.isFetching}>
              Search
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {results.isError && <Alert type="error" showIcon title={(results.error as Error).message} />}

      {terms && (
        <Card>
          <UserList
            users={results.data ?? []}
            loading={results.isFetching}
            empty="Nobody matched that name."
            renderAction={(user) => (user.id ? <FollowButton userId={user.id} /> : null)}
          />
        </Card>
      )}
    </Space>
  )
}
