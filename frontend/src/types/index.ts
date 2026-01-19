export interface User {
  id: string
  email: string
  role: 'user' | 'premium' | 'admin'
  created_at: string
  updated_at: string
}

export interface Subscription {
  id: string
  user_id: string
  plan: 'free' | 'starter' | 'pro' | 'unlimited'
  status: 'active' | 'past_due' | 'canceled' | 'trialing'
  download_limit_gb: number
  concurrent_limit: number
  retention_days: number
  created_at: string
}

export interface UsageStats {
  used_gb: number
  limit_gb: number
  active_torrents: number
  concurrent_limit: number
  plan: string
}

export interface TorrentFile {
  path: string
  size: number
  progress: number
  priority: number
}

export interface Torrent {
  id: string
  user_id: string
  info_hash: string
  name: string
  magnet_uri?: string
  status: 'pending' | 'downloading' | 'seeding' | 'completed' | 'failed' | 'paused'
  total_size: number
  downloaded_size: number
  uploaded_size: number
  download_speed: number
  upload_speed: number
  progress: number
  peers: number
  seeds: number
  files?: TorrentFile[]
  zip_path?: string
  zip_size?: number
  error_message?: string
  started_at?: string
  completed_at?: string
  expires_at?: string
  created_at: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
}

export interface MeResponse {
  user: User
  subscription: Subscription | null
  usage: UsageStats
}

export interface TorrentListResponse {
  torrents: Torrent[]
  total_count: number
  page: number
  page_size: number
}

export interface ApiError {
  error: string
  code?: string
  details?: string
}
