import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface Notification { id: string; title: string; message: string; entity_type: string; entity_id: string; read_at: string | null; created_at: string }

export const listNotifications = (page = 1, limit = 20, unread = false) => apiRequest<ApiEnvelope<Notification[]>>(`/notifications?page=${page}&limit=${limit}${unread ? '&unread=true' : ''}`)
export const markNotificationRead = (id: string) => apiRequest<ApiEnvelope<{ message: string }>>(`/notifications/${id}/read`, { method: 'POST' })
export const markAllNotificationsRead = () => apiRequest<ApiEnvelope<{ message: string }>>('/notifications/read-all', { method: 'POST' })