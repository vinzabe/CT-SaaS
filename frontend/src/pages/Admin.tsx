import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  Users, 
  Download, 
  Activity, 
  Trash2, 
  RefreshCw,
  ChevronLeft,
  ChevronRight
} from 'lucide-react'
import toast from 'react-hot-toast'
import { Layout } from '../components/Layout'
import { adminApi } from '../lib/api'
import { formatBytes, formatTimeAgo, cn } from '../lib/utils'

type TabType = 'overview' | 'users' | 'torrents'

export function AdminPage() {
  const [activeTab, setActiveTab] = useState<TabType>('overview')
  const [userPage, setUserPage] = useState(1)
  const [torrentPage, setTorrentPage] = useState(1)
  const queryClient = useQueryClient()

  const { data: stats } = useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: adminApi.getStats,
    refetchInterval: 5000,
  })

  const { data: usersData, isLoading: usersLoading } = useQuery({
    queryKey: ['admin', 'users', userPage],
    queryFn: () => adminApi.getUsers(userPage, 20),
    enabled: activeTab === 'users',
  })

  const { data: torrentsData, isLoading: torrentsLoading } = useQuery({
    queryKey: ['admin', 'torrents', torrentPage],
    queryFn: () => adminApi.getAllTorrents(torrentPage, 20),
    enabled: activeTab === 'torrents',
  })

  const deleteUserMutation = useMutation({
    mutationFn: (id: string) => adminApi.deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
      toast.success('User deleted')
    },
    onError: () => toast.error('Failed to delete user'),
  })

  const deleteTorrentMutation = useMutation({
    mutationFn: (id: string) => adminApi.deleteTorrent(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'torrents'] })
      toast.success('Torrent deleted')
    },
    onError: () => toast.error('Failed to delete torrent'),
  })

  const cleanupMutation = useMutation({
    mutationFn: adminApi.cleanup,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['admin'] })
      toast.success(`Cleaned up ${data.removed} expired torrents`)
    },
    onError: () => toast.error('Cleanup failed'),
  })

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Admin Panel</h1>
            <p className="text-gray-500">Manage users and monitor the platform</p>
          </div>
          <button
            onClick={() => cleanupMutation.mutate()}
            disabled={cleanupMutation.isPending}
            className="btn-secondary"
          >
            <RefreshCw className={cn('w-4 h-4 mr-2', cleanupMutation.isPending && 'animate-spin')} />
            Cleanup Expired
          </button>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 bg-gray-100 p-1 rounded-lg w-fit">
          {[
            { id: 'overview' as TabType, label: 'Overview', icon: Activity },
            { id: 'users' as TabType, label: 'Users', icon: Users },
            { id: 'torrents' as TabType, label: 'Torrents', icon: Download },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-md transition-colors',
                activeTab === tab.id
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              )}
            >
              <tab.icon className="w-4 h-4" />
              {tab.label}
            </button>
          ))}
        </div>

        {/* Overview Tab */}
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Stats Grid */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="card p-6">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center">
                    <Users className="w-6 h-6 text-blue-600" />
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Total Users</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {stats?.users?.total || 0}
                    </p>
                  </div>
                </div>
              </div>

              <div className="card p-6">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-green-100 rounded-xl flex items-center justify-center">
                    <Download className="w-6 h-6 text-green-600" />
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Total Torrents</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {stats?.torrents?.total || 0}
                    </p>
                  </div>
                </div>
              </div>

              <div className="card p-6">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-purple-100 rounded-xl flex items-center justify-center">
                    <Activity className="w-6 h-6 text-purple-600" />
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Active Downloads</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {stats?.torrents?.downloading || 0}
                    </p>
                  </div>
                </div>
              </div>

              <div className="card p-6">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-orange-100 rounded-xl flex items-center justify-center">
                    <Activity className="w-6 h-6 text-orange-600" />
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Download Speed</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatBytes(stats?.bandwidth?.download_speed_bps || 0)}/s
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* Activity breakdown */}
            <div className="card p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Torrent Status</h3>
              <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                {[
                  { label: 'Downloading', value: stats?.torrents?.downloading || 0, color: 'bg-blue-500' },
                  { label: 'Seeding', value: stats?.torrents?.seeding || 0, color: 'bg-green-500' },
                  { label: 'Completed', value: stats?.torrents?.completed || 0, color: 'bg-emerald-500' },
                  { label: 'Active', value: stats?.torrents?.active || 0, color: 'bg-purple-500' },
                  { label: 'Total', value: stats?.torrents?.total || 0, color: 'bg-gray-500' },
                ].map((item) => (
                  <div key={item.label} className="text-center">
                    <div className={cn('w-3 h-3 rounded-full mx-auto mb-2', item.color)} />
                    <p className="text-2xl font-bold text-gray-900">{item.value}</p>
                    <p className="text-sm text-gray-500">{item.label}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Users Tab */}
        {activeTab === 'users' && (
          <div className="space-y-4">
            <div className="card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-gray-50 border-b border-gray-200">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Plan</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Role</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Joined</th>
                      <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {usersLoading ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-8 text-center text-gray-500">
                          Loading...
                        </td>
                      </tr>
                    ) : usersData?.users?.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-8 text-center text-gray-500">
                          No users found
                        </td>
                      </tr>
                    ) : (
                      usersData?.users?.map((user: any) => (
                        <tr key={user.id} className="hover:bg-gray-50">
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-3">
                              <div className="w-8 h-8 bg-gray-100 rounded-full flex items-center justify-center">
                                <span className="text-sm font-medium text-gray-600">
                                  {user.email[0].toUpperCase()}
                                </span>
                              </div>
                              <span className="text-sm text-gray-900">{user.email}</span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <span className="px-2 py-1 text-xs font-medium bg-gray-100 text-gray-700 rounded capitalize">
                              {user.subscription?.plan || 'free'}
                            </span>
                          </td>
                          <td className="px-6 py-4">
                            <span className={cn(
                              'px-2 py-1 text-xs font-medium rounded capitalize',
                              user.role === 'admin' ? 'bg-purple-100 text-purple-700' : 'bg-gray-100 text-gray-700'
                            )}>
                              {user.role}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">
                            {formatTimeAgo(user.created_at)}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => {
                                if (confirm('Are you sure you want to delete this user?')) {
                                  deleteUserMutation.mutate(user.id)
                                }
                              }}
                              className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                              title="Delete user"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              {usersData?.total_count > 20 && (
                <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200">
                  <p className="text-sm text-gray-500">
                    Showing {((userPage - 1) * 20) + 1} to {Math.min(userPage * 20, usersData.total_count)} of {usersData.total_count}
                  </p>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setUserPage((p) => Math.max(1, p - 1))}
                      disabled={userPage === 1}
                      className="btn-secondary px-3 py-1"
                    >
                      <ChevronLeft className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => setUserPage((p) => p + 1)}
                      disabled={userPage * 20 >= usersData.total_count}
                      className="btn-secondary px-3 py-1"
                    >
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Torrents Tab */}
        {activeTab === 'torrents' && (
          <div className="space-y-4">
            <div className="card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-gray-50 border-b border-gray-200">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Torrent</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Size</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Progress</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Added</th>
                      <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {torrentsLoading ? (
                      <tr>
                        <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                          Loading...
                        </td>
                      </tr>
                    ) : torrentsData?.torrents?.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                          No torrents found
                        </td>
                      </tr>
                    ) : (
                      torrentsData?.torrents?.map((torrent: any) => (
                        <tr key={torrent.id} className="hover:bg-gray-50">
                          <td className="px-6 py-4">
                            <p className="text-sm text-gray-900 truncate max-w-xs" title={torrent.name}>
                              {torrent.name || 'Loading...'}
                            </p>
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">
                            {formatBytes(torrent.total_size)}
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-2">
                              <div className="w-24 h-2 bg-gray-200 rounded-full overflow-hidden">
                                <div
                                  className="h-full bg-primary-600 rounded-full"
                                  style={{ width: `${torrent.progress}%` }}
                                />
                              </div>
                              <span className="text-xs text-gray-500">
                                {torrent.progress?.toFixed(1)}%
                              </span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <span className={cn(
                              'px-2 py-1 text-xs font-medium rounded capitalize',
                              torrent.status === 'completed' && 'bg-green-100 text-green-700',
                              torrent.status === 'downloading' && 'bg-blue-100 text-blue-700',
                              torrent.status === 'failed' && 'bg-red-100 text-red-700',
                              torrent.status === 'paused' && 'bg-yellow-100 text-yellow-700',
                              !['completed', 'downloading', 'failed', 'paused'].includes(torrent.status) && 'bg-gray-100 text-gray-700'
                            )}>
                              {torrent.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">
                            {formatTimeAgo(torrent.created_at)}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => {
                                if (confirm('Delete this torrent?')) {
                                  deleteTorrentMutation.mutate(torrent.id)
                                }
                              }}
                              className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                              title="Delete"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              {torrentsData?.total_count > 20 && (
                <div className="flex items-center justify-between px-6 py-3 border-t border-gray-200">
                  <p className="text-sm text-gray-500">
                    Showing {((torrentPage - 1) * 20) + 1} to {Math.min(torrentPage * 20, torrentsData.total_count)} of {torrentsData.total_count}
                  </p>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setTorrentPage((p) => Math.max(1, p - 1))}
                      disabled={torrentPage === 1}
                      className="btn-secondary px-3 py-1"
                    >
                      <ChevronLeft className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => setTorrentPage((p) => p + 1)}
                      disabled={torrentPage * 20 >= torrentsData.total_count}
                      className="btn-secondary px-3 py-1"
                    >
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </Layout>
  )
}
