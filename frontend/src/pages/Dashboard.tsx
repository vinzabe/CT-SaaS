import { useState, useCallback, useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, RefreshCw, Search, Download, Zap, HardDrive, Wifi, WifiOff } from 'lucide-react'
import { Layout } from '../components/Layout'
import { TorrentCard } from '../components/TorrentCard'
import { AddTorrentModal } from '../components/AddTorrentModal'
import { torrentsApi, authApi } from '../lib/api'
import { useAuthStore } from '../lib/store'
import { formatBytes } from '../lib/utils'
import { useSSE, TransformedTorrentUpdate } from '../hooks/useSSE'
import type { Torrent } from '../types'

export function DashboardPage() {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const { setUser } = useAuthStore()
  const queryClient = useQueryClient()

  useQuery({
    queryKey: ['me'],
    queryFn: async () => {
      const data = await authApi.me()
      setUser(data.user, data.subscription, data.usage)
      return data
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  // Initial load and fallback polling (reduced frequency when SSE is connected)
  const { data: torrentsData, isLoading, refetch } = useQuery({
    queryKey: ['torrents'],
    queryFn: () => torrentsApi.list(1, 50),
    refetchInterval: 30000, // Reduced to 30s since SSE handles real-time updates
  })

  // Handle SSE updates by merging with existing data
  const handleSSEUpdate = useCallback((sseUpdates: TransformedTorrentUpdate[]) => {
    queryClient.setQueryData(['torrents'], (oldData: { torrents: Torrent[] } | undefined) => {
      if (!oldData) return oldData

      const updatedTorrents = oldData.torrents.map((torrent) => {
        const update = sseUpdates.find((u) => u.id === torrent.id || u.info_hash === torrent.info_hash)
        if (update) {
          return {
            ...torrent,
            status: update.status,
            progress: update.progress,
            downloaded_size: update.downloaded_size,
            uploaded_size: update.uploaded_size,
            download_speed: update.download_speed,
            upload_speed: update.upload_speed,
            peers: update.peers,
            seeds: update.seeds,
            name: update.name || torrent.name,
            total_size: update.total_size || torrent.total_size,
            files: update.files || torrent.files,
            error_message: update.error_message,
          }
        }
        return torrent
      })

      return { ...oldData, torrents: updatedTorrents }
    })
  }, [queryClient])

  // SSE connection for real-time updates
  const { status: sseStatus, isConnected } = useSSE({
    onTorrentsUpdate: handleSSEUpdate,
    enabled: true,
  })

  const torrents = torrentsData?.torrents || []
  const filteredTorrents = useMemo(() => 
    torrents.filter((t) =>
      t.name?.toLowerCase().includes(searchQuery.toLowerCase())
    ),
    [torrents, searchQuery]
  )

  // Stats calculations
  const stats = useMemo(() => ({
    activeTorrents: torrents.filter((t) => 
      t.status === 'downloading' || t.status === 'pending'
    ).length,
    completedTorrents: torrents.filter((t) => 
      t.status === 'completed' || t.status === 'seeding'
    ).length,
    totalDownloadSpeed: torrents.reduce((acc, t) => acc + (t.download_speed || 0), 0),
    totalSize: torrents.reduce((acc, t) => acc + (t.total_size || 0), 0),
  }), [torrents])

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
            <div className="flex items-center gap-2">
              <p className="text-gray-500">Manage your torrent downloads</p>
              {/* SSE Connection Status */}
              <span 
                className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${
                  isConnected 
                    ? 'bg-green-100 text-green-700' 
                    : sseStatus === 'connecting'
                    ? 'bg-yellow-100 text-yellow-700'
                    : 'bg-gray-100 text-gray-600'
                }`}
                title={isConnected ? 'Live updates active' : `Status: ${sseStatus}`}
              >
                {isConnected ? (
                  <>
                    <Wifi className="w-3 h-3" />
                    Live
                  </>
                ) : (
                  <>
                    <WifiOff className="w-3 h-3" />
                    {sseStatus === 'connecting' ? 'Connecting...' : 'Offline'}
                  </>
                )}
              </span>
            </div>
          </div>
          <button
            onClick={() => setIsAddModalOpen(true)}
            className="btn-primary"
          >
            <Plus className="w-5 h-5 mr-2" />
            Add Torrent
          </button>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
                <Download className="w-5 h-5 text-blue-600" />
              </div>
              <div>
                <p className="text-sm text-gray-500">Active</p>
                <p className="text-xl font-semibold text-gray-900">{stats.activeTorrents}</p>
              </div>
            </div>
          </div>

          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center">
                <Zap className="w-5 h-5 text-green-600" />
              </div>
              <div>
                <p className="text-sm text-gray-500">Speed</p>
                <p className="text-xl font-semibold text-gray-900">
                  {formatBytes(stats.totalDownloadSpeed)}/s
                </p>
              </div>
            </div>
          </div>

          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-purple-100 rounded-lg flex items-center justify-center">
                <HardDrive className="w-5 h-5 text-purple-600" />
              </div>
              <div>
                <p className="text-sm text-gray-500">Total Size</p>
                <p className="text-xl font-semibold text-gray-900">
                  {formatBytes(stats.totalSize)}
                </p>
              </div>
            </div>
          </div>

          <div className="card p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-orange-100 rounded-lg flex items-center justify-center">
                <Download className="w-5 h-5 text-orange-600" />
              </div>
              <div>
                <p className="text-sm text-gray-500">Completed</p>
                <p className="text-xl font-semibold text-gray-900">{stats.completedTorrents}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Search and filters */}
        <div className="flex items-center gap-4">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder="Search torrents..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="input pl-10"
            />
          </div>
          <button
            onClick={() => refetch()}
            className="btn-secondary"
            title="Refresh"
          >
            <RefreshCw className="w-5 h-5" />
          </button>
        </div>

        {/* Torrents list */}
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="card p-4 animate-pulse">
                <div className="flex items-start gap-4">
                  <div className="w-12 h-12 bg-gray-200 rounded-lg" />
                  <div className="flex-1 space-y-3">
                    <div className="h-4 bg-gray-200 rounded w-1/2" />
                    <div className="h-3 bg-gray-200 rounded w-1/3" />
                    <div className="h-2 bg-gray-200 rounded w-full" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : filteredTorrents.length === 0 ? (
          <div className="card p-12 text-center">
            <Download className="w-12 h-12 mx-auto text-gray-300 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              {searchQuery ? 'No torrents found' : 'No torrents yet'}
            </h3>
            <p className="text-gray-500 mb-6">
              {searchQuery
                ? 'Try a different search term'
                : 'Add your first torrent to get started'}
            </p>
            {!searchQuery && (
              <button
                onClick={() => setIsAddModalOpen(true)}
                className="btn-primary"
              >
                <Plus className="w-5 h-5 mr-2" />
                Add Torrent
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-4">
            {filteredTorrents.map((torrent) => (
              <TorrentCard key={torrent.id} torrent={torrent} />
            ))}
          </div>
        )}
      </div>

      <AddTorrentModal
        isOpen={isAddModalOpen}
        onClose={() => setIsAddModalOpen(false)}
      />
    </Layout>
  )
}
