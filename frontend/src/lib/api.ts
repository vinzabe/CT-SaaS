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

// Track if we're currently refreshing to avoid multiple refresh attempts
let isRefreshing = false
let failedQueue: Array<{ resolve: (token: string) => void; reject: (error: unknown) => void }> = []

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token!)
    }
  })
  failedQueue = []
}

// Response interceptor to handle token refresh
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    const originalRequest = error.config as typeof error.config & { _retry?: boolean }
    
    // Handle 401 errors - only if we have a token (user was logged in)
    if (error.response?.status === 401 && useAuthStore.getState().accessToken) {
      // Don't retry if we already tried
      if (originalRequest?._retry) {
        return Promise.reject(error)
      }

      // Try to refresh token
      if (isRefreshing) {
        // Queue this request while refresh is in progress
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        }).then((token) => {
          if (originalRequest) {
            originalRequest.headers.Authorization = `Bearer ${token}`
            return api(originalRequest)
          }
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        const refreshToken = useAuthStore.getState().refreshToken
        if (refreshToken) {
          const response = await axios.post<AuthResponse>('/api/v1/auth/refresh', {
            refresh_token: refreshToken,
          })
          
          const newToken = response.data.access_token
          useAuthStore.getState().setTokens(newToken, response.data.refresh_token)
          
          processQueue(null, newToken)
          
          if (originalRequest) {
            originalRequest.headers.Authorization = `Bearer ${newToken}`
            return api(originalRequest)
          }
        } else {
          throw new Error('No refresh token')
        }
      } catch (refreshError) {
        processQueue(refreshError, null)
        // Only logout if refresh actually failed, and only if on a protected page
        if (window.location.pathname.startsWith('/dashboard') || 
            window.location.pathname.startsWith('/admin')) {
          useAuthStore.getState().logout()
          // Use replace to avoid adding to history
          window.location.replace('/login')
        }
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
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
