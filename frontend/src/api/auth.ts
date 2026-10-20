import { apiRequest } from './client'
import type { AuthSession, AuthUser } from '../store/authStore'

interface LoginEnvelope {
  success: boolean
  data: AuthSession
}

export async function login(email: string, password: string): Promise<AuthSession> {
  const response = await apiRequest<LoginEnvelope>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })

  if (!response.success || !response.data?.token || !response.data.user) {
    throw new Error('The server returned an invalid login response.')
  }

  const user = response.data.user as AuthUser
  if (!user.role_name) throw new Error('Your account does not have a role assigned.')
  return { token: response.data.token, user }
}
