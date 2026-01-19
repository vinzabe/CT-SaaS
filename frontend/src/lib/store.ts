import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User, Subscription, UsageStats } from '../types'

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  user: User | null
  subscription: Subscription | null
  usage: UsageStats | null
  isAuthenticated: boolean
  
  setTokens: (accessToken: string, refreshToken: string) => void
  setUser: (user: User, subscription: Subscription | null, usage: UsageStats) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      subscription: null,
      usage: null,
      isAuthenticated: false,
      
      setTokens: (accessToken, refreshToken) => set({
        accessToken,
        refreshToken,
        isAuthenticated: true,
      }),
      
      setUser: (user, subscription, usage) => set({
        user,
        subscription,
        usage,
        isAuthenticated: true,
      }),
      
      logout: () => set({
        accessToken: null,
        refreshToken: null,
        user: null,
        subscription: null,
        usage: null,
        isAuthenticated: false,
      }),
    }),
    {
      name: 'grants-torrent-auth',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        isAuthenticated: state.isAuthenticated,
        user: state.user,
        subscription: state.subscription,
        usage: state.usage,
      }),
    }
  )
)
