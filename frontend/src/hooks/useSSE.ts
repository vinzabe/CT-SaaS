import { useEffect, useRef, useCallback, useState } from 'react'
import { useAuthStore } from '../lib/store'

// SSE event types from backend
export interface SSETorrentUpdate {
  ID: string
  InfoHash: string
  Status: string
  Progress: number
  Downloaded: number
  Uploaded: number
  DownloadSpeed: number
  UploadSpeed: number
  Peers: number
  Seeds: number
  Name: string
  TotalSize: number
  Files?: Array<{
    Path: string
    Size: number
    Progress: number
    Priority: number
  }>
  Error?: string
}

// Transform backend format to frontend Torrent format
function transformTorrentUpdate(update: SSETorrentUpdate) {
  return {
    id: update.ID,
    info_hash: update.InfoHash,
    status: update.Status.toLowerCase() as 'pending' | 'downloading' | 'completed' | 'failed' | 'paused' | 'seeding',
    progress: update.Progress,
    downloaded_size: update.Downloaded,
    uploaded_size: update.Uploaded,
    download_speed: update.DownloadSpeed,
    upload_speed: update.UploadSpeed,
    peers: update.Peers,
    seeds: update.Seeds,
    name: update.Name,
    total_size: update.TotalSize,
    files: update.Files?.map(f => ({
      path: f.Path,
      size: f.Size,
      progress: f.Progress,
      priority: f.Priority,
    })),
    error_message: update.Error,
  }
}

export type TransformedTorrentUpdate = ReturnType<typeof transformTorrentUpdate>

interface UseSSEOptions {
  onTorrentsUpdate?: (torrents: TransformedTorrentUpdate[]) => void
  onConnected?: () => void
  onError?: (error: Event) => void
  onHeartbeat?: (time: number) => void
  enabled?: boolean
  reconnectInterval?: number
}

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

export function useSSE({
  onTorrentsUpdate,
  onConnected,
  onError,
  onHeartbeat,
  enabled = true,
  reconnectInterval = 5000,
}: UseSSEOptions = {}) {
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const accessToken = useAuthStore((state) => state.accessToken)
  const [status, setStatus] = useState<ConnectionStatus>('disconnected')
  const [lastHeartbeat, setLastHeartbeat] = useState<number | null>(null)

  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
  }, [])

  const connect = useCallback(() => {
    if (!accessToken || !enabled) {
      setStatus('disconnected')
      return
    }

    // Clean up existing connection
    cleanup()

    setStatus('connecting')

    // Create EventSource with token as query param (SSE doesn't support headers)
    const eventSource = new EventSource(`/api/v1/events?token=${encodeURIComponent(accessToken)}`)
    eventSourceRef.current = eventSource

    eventSource.addEventListener('connected', () => {
      setStatus('connected')
      onConnected?.()
    })

    eventSource.addEventListener('torrents', (event) => {
      try {
        const rawTorrents: SSETorrentUpdate[] = JSON.parse(event.data)
        const torrents = rawTorrents.map(transformTorrentUpdate)
        onTorrentsUpdate?.(torrents)
      } catch (e) {
        console.error('Failed to parse SSE torrents data:', e)
      }
    })

    eventSource.addEventListener('heartbeat', (event) => {
      try {
        const data = JSON.parse(event.data)
        const time = data.time as number
        setLastHeartbeat(time)
        onHeartbeat?.(time)
      } catch (e) {
        // Heartbeat parsing failed, ignore
      }
    })

    eventSource.addEventListener('timeout', () => {
      // Server closed connection after timeout, reconnect
      cleanup()
      setStatus('disconnected')
      reconnectTimeoutRef.current = setTimeout(connect, 1000)
    })

    eventSource.onerror = (event) => {
      setStatus('error')
      onError?.(event)
      cleanup()
      // Reconnect after interval
      reconnectTimeoutRef.current = setTimeout(connect, reconnectInterval)
    }
  }, [accessToken, enabled, cleanup, onConnected, onTorrentsUpdate, onHeartbeat, onError, reconnectInterval])

  // Connect on mount and when dependencies change
  useEffect(() => {
    connect()
    return cleanup
  }, [connect, cleanup])

  // Disconnect when token is cleared (logout)
  useEffect(() => {
    if (!accessToken) {
      cleanup()
      setStatus('disconnected')
    }
  }, [accessToken, cleanup])

  const disconnect = useCallback(() => {
    cleanup()
    setStatus('disconnected')
  }, [cleanup])

  const reconnect = useCallback(() => {
    cleanup()
    connect()
  }, [cleanup, connect])

  return {
    status,
    lastHeartbeat,
    disconnect,
    reconnect,
    isConnected: status === 'connected',
  }
}
