import axios, { AxiosError } from 'axios'
import type { AuthResponse, MeResponse, Torrent, TorrentListResponse, ApiError } from '../types'
import { useAuthStore } from './store'

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor to handle token refresh
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    const originalRequest = error.config
    
    // Handle 401 errors
    if (error.response?.status === 401) {
      // If token expired, try to refresh
      if (error.response?.data?.code === 'TOKEN_EXPIRED') {
        try {
          const refreshToken = useAuthStore.getState().refreshToken
          if (refreshToken) {
            const response = await axios.post<AuthResponse>('/api/v1/auth/refresh', {
              refresh_token: refreshToken,
            })
            
            useAuthStore.getState().setTokens(
              response.data.access_token,
              response.data.refresh_token
            )
            
            if (originalRequest) {
              originalRequest.headers.Authorization = `Bearer ${response.data.access_token}`
              return api(originalRequest)
            }
          }
        } catch {
          useAuthStore.getState().logout()
          window.location.href = '/login'
        }
      } else {
        // For any other 401 (missing header, invalid token), logout and redirect
        useAuthStore.getState().logout()
        window.location.href = '/login'
      }
    }
    
    return Promise.reject(error)
  }
)

// Auth API
export const authApi = {
  register: async (email: string, password: string) => {
    const response = await api.post<AuthResponse>('/auth/register', { email, password })
    return response.data
  },
  
  login: async (email: string, password: string) => {
    const response = await api.post<AuthResponse>('/auth/login', { email, password })
    return response.data
  },
  
  logout: async (refreshToken: string) => {
    await api.post('/auth/logout', { refresh_token: refreshToken })
  },
  
  me: async () => {
    const response = await api.get<MeResponse>('/auth/me')
    return response.data
  },
}

// Torrents API
export const torrentsApi = {
  list: async (page = 1, pageSize = 20) => {
    const response = await api.get<TorrentListResponse>('/torrents', {
      params: { page, page_size: pageSize },
    })
    return response.data
  },
  
  get: async (id: string) => {
    const response = await api.get<Torrent>(`/torrents/${id}`)
    return response.data
  },
  
  addMagnet: async (magnetUri: string) => {
    const response = await api.post<Torrent>('/torrents', { magnet_uri: magnetUri })
    return response.data
  },
  
  addUrl: async (torrentUrl: string) => {
    const response = await api.post<Torrent>('/torrents', { torrent_url: torrentUrl })
    return response.data
  },
  
  upload: async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    const response = await api.post<Torrent>('/torrents/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return response.data
  },
  
  delete: async (id: string, deleteFiles = true) => {
    await api.delete(`/torrents/${id}`, { params: { delete_files: deleteFiles } })
  },
  
  pause: async (id: string) => {
    await api.post(`/torrents/${id}/pause`)
  },
  
  resume: async (id: string) => {
    await api.post(`/torrents/${id}/resume`)
  },
  
  createDownloadToken: async (torrentId: string, filePath: string, useZip = false) => {
    const response = await api.post<{ token: string; download_url: string; expires_in: number; is_zip: boolean }>(
      `/torrents/${torrentId}/token`,
      { file_path: filePath, use_zip: useZip }
    )
    return response.data
  },
}

// Admin API
export const adminApi = {
  getUsers: async (page = 1, pageSize = 20) => {
    const response = await api.get('/admin/users', { params: { page, page_size: pageSize } })
    return response.data
  },
  
  getUser: async (id: string) => {
    const response = await api.get(`/admin/users/${id}`)
    return response.data
  },
  
  updateUser: async (id: string, data: { role?: string; plan?: string }) => {
    await api.patch(`/admin/users/${id}`, data)
  },
  
  deleteUser: async (id: string) => {
    await api.delete(`/admin/users/${id}`)
  },
  
  getAllTorrents: async (page = 1, pageSize = 20) => {
    const response = await api.get('/admin/torrents', { params: { page, page_size: pageSize } })
    return response.data
  },
  
  deleteTorrent: async (id: string) => {
    await api.delete(`/admin/torrents/${id}`)
  },
  
  getStats: async () => {
    const response = await api.get('/admin/stats')
    return response.data
  },
  
  cleanup: async () => {
    const response = await api.post('/admin/cleanup')
    return response.data
  },
}

export default api
