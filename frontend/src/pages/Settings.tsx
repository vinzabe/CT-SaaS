import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { User, CreditCard, Bell, Shield, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Layout } from '../components/Layout'
import { useAuthStore } from '../lib/store'
import api from '../lib/api'

export function SettingsPage() {
  const { user, subscription } = useAuthStore()
  const [activeTab, setActiveTab] = useState<'account' | 'subscription' | 'notifications'>('account')

  const portalMutation = useMutation({
    mutationFn: async () => {
      const response = await api.post('/subscription/portal')
      return response.data
    },
    onSuccess: (data) => {
      if (data.url) {
        window.location.href = data.url
      }
    },
    onError: () => toast.error('Failed to open billing portal'),
  })

  const tabs = [
    { id: 'account' as const, label: 'Account', icon: User },
    { id: 'subscription' as const, label: 'Subscription', icon: CreditCard },
    { id: 'notifications' as const, label: 'Notifications', icon: Bell },
  ]

  return (
    <Layout>
      <div className="max-w-4xl mx-auto space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
          <p className="text-gray-500">Manage your account and preferences</p>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 bg-gray-100 p-1 rounded-lg w-fit">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                activeTab === tab.id
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              {tab.label}
            </button>
          ))}
        </div>

        {/* Account Tab */}
        {activeTab === 'account' && (
          <div className="card p-6 space-y-6">
            {user?.role === 'demo' && (
              <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p className="text-sm font-medium text-yellow-800">Demo Account</p>
                <p className="text-xs text-yellow-600 mt-1">
                  This is a demo account. Downloads are automatically deleted after 24 hours.
                  Account settings cannot be changed.
                </p>
              </div>
            )}
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Account Information</h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
                  <input
                    type="email"
                    value={user?.email || ''}
                    disabled
                    className="input bg-gray-50 cursor-not-allowed"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Role</label>
                  <input
                    type="text"
                    value={user?.role === 'demo' ? 'Demo User' : (user?.role || 'user')}
                    disabled
                    className="input bg-gray-50 cursor-not-allowed capitalize"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Member Since</label>
                  <input
                    type="text"
                    value={user?.created_at ? new Date(user.created_at).toLocaleDateString() : '-'}
                    disabled
                    className="input bg-gray-50 cursor-not-allowed"
                  />
                </div>
              </div>
            </div>

            <div className="pt-4 border-t border-gray-200">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Security</h3>
              <div className="flex items-center gap-3 p-4 bg-green-50 border border-green-200 rounded-lg">
                <Shield className="w-5 h-5 text-green-600" />
                <div>
                  <p className="text-sm font-medium text-green-800">Post-Quantum Security Enabled</p>
                  <p className="text-xs text-green-600">Your account is protected with ML-DSA-65 signatures</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Subscription Tab */}
        {activeTab === 'subscription' && (
          <div className="card p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Current Plan</h3>
              <div className="p-4 bg-primary-50 border border-primary-200 rounded-lg">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-lg font-semibold text-primary-900 capitalize">
                      {subscription?.plan || 'Free'} Plan
                    </p>
                    <p className="text-sm text-primary-700">
                      {subscription?.status === 'active' ? 'Active' : 'Inactive'}
                    </p>
                  </div>
                  {subscription?.plan !== 'free' && (
                    <button
                      onClick={() => portalMutation.mutate()}
                      disabled={portalMutation.isPending}
                      className="btn-secondary"
                    >
                      {portalMutation.isPending ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        'Manage Billing'
                      )}
                    </button>
                  )}
                </div>
              </div>
            </div>

            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Plan Limits</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="p-4 bg-gray-50 rounded-lg">
                  <p className="text-sm text-gray-500">Monthly Download</p>
                  <p className="text-xl font-semibold text-gray-900">
                    {subscription?.download_limit_gb === -1 ? 'Unlimited' : `${subscription?.download_limit_gb || 2} GB`}
                  </p>
                </div>
                <div className="p-4 bg-gray-50 rounded-lg">
                  <p className="text-sm text-gray-500">Concurrent Downloads</p>
                  <p className="text-xl font-semibold text-gray-900">
                    {subscription?.concurrent_limit || 1}
                  </p>
                </div>
                <div className="p-4 bg-gray-50 rounded-lg">
                  <p className="text-sm text-gray-500">File Retention</p>
                  <p className="text-xl font-semibold text-gray-900">
                    {subscription?.retention_days || 1} days
                  </p>
                </div>
              </div>
            </div>

            {subscription?.plan === 'free' && (
              <div className="pt-4 border-t border-gray-200">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Upgrade Your Plan</h3>
                <p className="text-gray-600 mb-4">
                  Get more downloads, concurrent torrents, and longer file retention with a premium plan.
                </p>
                <a href="/" className="btn-primary inline-flex">
                  View Plans
                </a>
              </div>
            )}
          </div>
        )}

        {/* Notifications Tab */}
        {activeTab === 'notifications' && (
          <div className="card p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Email Notifications</h3>
              <div className="space-y-4">
                <label className="flex items-center justify-between p-4 bg-gray-50 rounded-lg cursor-pointer">
                  <div>
                    <p className="font-medium text-gray-900">Download Complete</p>
                    <p className="text-sm text-gray-500">Get notified when your downloads finish</p>
                  </div>
                  <input
                    type="checkbox"
                    defaultChecked
                    className="w-5 h-5 text-primary-600 rounded focus:ring-primary-500"
                  />
                </label>
                <label className="flex items-center justify-between p-4 bg-gray-50 rounded-lg cursor-pointer">
                  <div>
                    <p className="font-medium text-gray-900">Usage Alerts</p>
                    <p className="text-sm text-gray-500">Get notified when approaching usage limits</p>
                  </div>
                  <input
                    type="checkbox"
                    defaultChecked
                    className="w-5 h-5 text-primary-600 rounded focus:ring-primary-500"
                  />
                </label>
                <label className="flex items-center justify-between p-4 bg-gray-50 rounded-lg cursor-pointer">
                  <div>
                    <p className="font-medium text-gray-900">Marketing Emails</p>
                    <p className="text-sm text-gray-500">Receive updates about new features and offers</p>
                  </div>
                  <input
                    type="checkbox"
                    className="w-5 h-5 text-primary-600 rounded focus:ring-primary-500"
                  />
                </label>
              </div>
            </div>

            <div className="pt-4">
              <button className="btn-primary">Save Preferences</button>
            </div>
          </div>
        )}
      </div>
    </Layout>
  )
}
