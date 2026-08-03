import { Navigate, Route, Routes } from 'react-router-dom'
import { AppLayout } from './components/AppLayout'
import { ProtectedRoute } from './auth/ProtectedRoute'
import { LoginPage } from './features/auth/LoginPage'
import { RegisterPage } from './features/auth/RegisterPage'
import { FeedPage } from './features/feed/FeedPage'
import { SearchPage } from './features/search/SearchPage'
import { FriendsPage } from './features/friends/FriendsPage'
import { ProfilePage } from './features/profile/ProfilePage'
import { DialogsPage } from './features/dialogs/DialogsPage'
import { DialogThreadPage } from './features/dialogs/DialogThreadPage'

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      <Route
        element={
          <ProtectedRoute>
            <AppLayout />
          </ProtectedRoute>
        }
      >
        <Route path="/feed" element={<FeedPage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/friends" element={<FriendsPage />} />
        <Route path="/me" element={<ProfilePage self />} />
        <Route path="/users/:id" element={<ProfilePage />} />
        <Route path="/dialogs" element={<DialogsPage />} />
        <Route path="/dialogs/:userId" element={<DialogThreadPage />} />
      </Route>

      <Route path="*" element={<Navigate to="/feed" replace />} />
    </Routes>
  )
}
