import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Cloud, Loader2, Eye, EyeOff, Check } from 'lucide-react'
import toast from 'react-hot-toast'
import { authApi } from '../lib/api'
import { useAuthStore } from '../lib/store'
import { cn } from '../lib/utils'

export function RegisterPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const navigate = useNavigate()
  const { setTokens, setUser } = useAuthStore()

  const registerMutation = useMutation({
    mutationFn: () => authApi.register(email, password),
    onSuccess: async (data) => {
      setTokens(data.access_token, data.refresh_token)
      
      const meData = await authApi.me()
      setUser(meData.user, meData.subscription, meData.usage)
      
      toast.success('Account created successfully!')
      navigate('/dashboard')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Registration failed')
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    if (password !== confirmPassword) {
      toast.error('Passwords do not match')
      return
    }
    
    registerMutation.mutate()
  }

  // Password strength checks
  const passwordChecks = [
    { label: 'At least 8 characters', valid: password.length >= 8 },
    { label: 'Contains uppercase letter', valid: /[A-Z]/.test(password) },
    { label: 'Contains lowercase letter', valid: /[a-z]/.test(password) },
    { label: 'Contains number', valid: /[0-9]/.test(password) },
  ]

  const allChecksPass = passwordChecks.every((c) => c.valid)

  return (
    <div className="min-h-screen bg-gradient-to-br from-primary-600 to-primary-800 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <Link to="/" className="inline-flex items-center gap-2">
            <Cloud className="w-12 h-12 text-white" />
            <span className="text-3xl font-bold text-white">Free Torrent</span>
          </Link>
          <p className="text-primary-200 mt-2">
            Create your free account
          </p>
        </div>

        {/* Card */}
        <div className="bg-white rounded-2xl shadow-xl p-8">
          <h2 className="text-2xl font-bold text-gray-900 text-center mb-6">
            Create an account
          </h2>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
                Email address
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="input"
                placeholder="you@example.com"
                required
                autoComplete="email"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
                Password
              </label>
              <div className="relative">
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="input pr-10"
                  placeholder="Create a password"
                  required
                  autoComplete="new-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                >
                  {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                </button>
              </div>
              
              {/* Password requirements */}
              {password && (
                <div className="mt-2 space-y-1">
                  {passwordChecks.map((check, i) => (
                    <div
                      key={i}
                      className={cn(
                        'flex items-center gap-2 text-xs',
                        check.valid ? 'text-green-600' : 'text-gray-400'
                      )}
                    >
                      <Check className={cn('w-3.5 h-3.5', !check.valid && 'opacity-0')} />
                      {check.label}
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 mb-1">
                Confirm password
              </label>
              <input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className={cn(
                  'input',
                  confirmPassword && password !== confirmPassword && 'border-red-300 focus:ring-red-500 focus:border-red-500'
                )}
                placeholder="Confirm your password"
                required
                autoComplete="new-password"
              />
              {confirmPassword && password !== confirmPassword && (
                <p className="text-xs text-red-600 mt-1">Passwords do not match</p>
              )}
            </div>

            <button
              type="submit"
              disabled={registerMutation.isPending || !allChecksPass || password !== confirmPassword}
              className="btn-primary w-full py-3"
            >
              {registerMutation.isPending ? (
                <>
                  <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                  Creating account...
                </>
              ) : (
                'Create account'
              )}
            </button>
          </form>

          <p className="text-center text-sm text-gray-600 mt-6">
            Already have an account?{' '}
            <Link to="/login" className="text-primary-600 hover:text-primary-700 font-medium">
              Sign in
            </Link>
          </p>
        </div>

        {/* Free plan info */}
        <div className="bg-white/10 backdrop-blur rounded-xl p-4 mt-6">
          <p className="text-white text-sm text-center">
            Start with our <strong>Free plan</strong>: 2GB/month, 1 concurrent download
          </p>
        </div>
      </div>
    </div>
  )
}
