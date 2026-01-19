import { Link, useLocation, useNavigate } from 'react-router-dom'
import { 
  Home, 
  Download, 
  Settings, 
  LogOut, 
  Shield,
  Cloud,
  Menu,
  X
} from 'lucide-react'
import { useState } from 'react'
import { useAuthStore } from '../lib/store'
import { authApi } from '../lib/api'
import { cn } from '../lib/utils'

interface LayoutProps {
  children: React.ReactNode
}

export function Layout({ children }: LayoutProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()
  const { user, subscription, usage, logout, refreshToken } = useAuthStore()

  const handleLogout = async () => {
    try {
      if (refreshToken) {
        await authApi.logout(refreshToken)
      }
    } finally {
      logout()
      navigate('/login')
    }
  }

  const navigation = [
    { name: 'Dashboard', href: '/dashboard', icon: Home },
    { name: 'My Torrents', href: '/dashboard/torrents', icon: Download },
    { name: 'Settings', href: '/dashboard/settings', icon: Settings },
  ]

  if (user?.role === 'admin') {
    navigation.push({ name: 'Admin', href: '/admin', icon: Shield })
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Mobile sidebar backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-gray-900/50 z-40 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <div
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 transform transition-transform lg:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex flex-col h-full">
          {/* Logo */}
          <div className="flex items-center justify-between h-16 px-4 border-b border-gray-200">
            <Link to="/dashboard" className="flex items-center gap-2">
              <Cloud className="w-8 h-8 text-primary-600" />
              <span className="text-xl font-bold text-gray-900">Grant's Torrent</span>
            </Link>
            <button
              className="lg:hidden p-2 rounded-lg hover:bg-gray-100"
              onClick={() => setSidebarOpen(false)}
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
            {navigation.map((item) => {
              const isActive = location.pathname === item.href ||
                (item.href !== '/dashboard' && location.pathname.startsWith(item.href))
              
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-lg transition-colors',
                    isActive
                      ? 'bg-primary-50 text-primary-700'
                      : 'text-gray-700 hover:bg-gray-100'
                  )}
                >
                  <item.icon className="w-5 h-5" />
                  {item.name}
                </Link>
              )
            })}
          </nav>

          {/* Usage Stats */}
          {usage && (
            <div className="p-4 border-t border-gray-200">
              <div className="text-sm text-gray-500 mb-2">Usage this month</div>
              <div className="space-y-2">
                <div>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-gray-700">Downloads</span>
                    <span className="text-gray-900 font-medium">
                      {usage.used_gb.toFixed(2)} / {usage.limit_gb === -1 ? '∞' : usage.limit_gb} GB
                    </span>
                  </div>
                  <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-primary-600 rounded-full transition-all"
                      style={{
                        width: usage.limit_gb === -1 
                          ? '10%' 
                          : `${Math.min(100, (usage.used_gb / usage.limit_gb) * 100)}%`
                      }}
                    />
                  </div>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Active torrents</span>
                  <span className="text-gray-900">
                    {usage.active_torrents} / {usage.concurrent_limit}
                  </span>
                </div>
              </div>
            </div>
          )}

          {/* User info */}
          <div className="p-4 border-t border-gray-200">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center">
                <span className="text-primary-700 font-medium">
                  {user?.email?.[0]?.toUpperCase()}
                </span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">
                  {user?.email}
                </p>
                <p className="text-xs text-gray-500 capitalize">
                  {subscription?.plan || 'Free'} Plan
                </p>
              </div>
              <button
                onClick={handleLogout}
                className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100"
                title="Logout"
              >
                <LogOut className="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Top bar */}
        <div className="sticky top-0 z-30 h-16 bg-white border-b border-gray-200 lg:hidden">
          <div className="flex items-center justify-between h-full px-4">
            <button
              onClick={() => setSidebarOpen(true)}
              className="p-2 rounded-lg hover:bg-gray-100"
            >
              <Menu className="w-6 h-6" />
            </button>
            <Link to="/dashboard" className="flex items-center gap-2">
              <Cloud className="w-6 h-6 text-primary-600" />
              <span className="font-bold text-gray-900">Grant's Torrent</span>
            </Link>
            <div className="w-10" />
          </div>
        </div>

        {/* Page content */}
        <main className="p-4 lg:p-8">
          {children}
        </main>
      </div>
    </div>
  )
}
