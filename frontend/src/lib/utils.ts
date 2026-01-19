import { clsx, type ClassValue } from 'clsx'

export function cn(...inputs: ClassValue[]) {
  return clsx(inputs)
}

export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

export function formatSpeed(bytesPerSecond: number): string {
  return `${formatBytes(bytesPerSecond)}/s`
}

export function formatProgress(progress: number): string {
  return `${Math.min(100, Math.max(0, progress)).toFixed(1)}%`
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.floor(seconds)}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${Math.floor(seconds % 60)}s`
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${minutes}m`
}

export function formatTimeAgo(date: string | Date): string {
  const now = new Date()
  const then = new Date(date)
  const seconds = Math.floor((now.getTime() - then.getTime()) / 1000)
  
  if (seconds < 60) return 'just now'
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`
  
  return then.toLocaleDateString()
}

export function getStatusColor(status: string): string {
  switch (status) {
    case 'downloading':
      return 'text-blue-600 bg-blue-100'
    case 'seeding':
      return 'text-green-600 bg-green-100'
    case 'completed':
      return 'text-green-600 bg-green-100'
    case 'paused':
      return 'text-yellow-600 bg-yellow-100'
    case 'failed':
      return 'text-red-600 bg-red-100'
    case 'pending':
      return 'text-gray-600 bg-gray-100'
    default:
      return 'text-gray-600 bg-gray-100'
  }
}

export function estimateTimeRemaining(totalSize: number, downloadedSize: number, speed: number): string {
  if (speed <= 0 || downloadedSize >= totalSize) return '--'
  const remaining = totalSize - downloadedSize
  const seconds = remaining / speed
  return formatDuration(seconds)
}
