import { useEffect, useRef, useCallback } from 'react'
import { useAuthStore } from '../lib/store'
import type { Torrent } from '../types'

interface UseSSEOptions {
  onTorrentsUpdate?: (torrents: Torrent[]) => void
  onError?: (error: Event) => void
  enabled?: boolean
}

export function useSSE({ onTorrentsUpdate, onError, enabled = true }: UseSSEOptions) {
  const eventSourceRef = useRef<EventSource | null>(null)
  const accessToken = useAuthStore((state) => state.accessToken)

  const connect = useCallback(() => {
    if (!accessToken || !enabled) return

    // Note: SSE with auth headers requires a different approach
    // For simplicity, we'll use polling instead or a custom implementation
    // In production, you'd use a library like eventsource-polyfill with headers
    
    const eventSource = new EventSource(`/api/v1/events?token=${accessToken}`)
    
    eventSource.addEventListener('torrents', (event) => {
      try {
        const torrents = JSON.parse(event.data)
        onTorrentsUpdate?.(torrents)
      } catch (e) {
        console.error('Failed to parse SSE data:', e)
      }
    })

    eventSource.addEventListener('error', (event) => {
      onError?.(event)
      eventSource.close()
      // Reconnect after 5 seconds
      setTimeout(connect, 5000)
    })

    eventSourceRef.current = eventSource
  }, [accessToken, enabled, onTorrentsUpdate, onError])

  useEffect(() => {
    connect()

    return () => {
      eventSourceRef.current?.close()
    }
  }, [connect])

  const disconnect = useCallback(() => {
    eventSourceRef.current?.close()
    eventSourceRef.current = null
  }, [])

  return { disconnect }
}
