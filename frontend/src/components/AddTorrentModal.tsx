import { useState, useRef } from 'react'
import { X, Link as LinkIcon, Upload, Loader2 } from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import toast from 'react-hot-toast'
import { torrentsApi } from '../lib/api'
import { cn } from '../lib/utils'

interface AddTorrentModalProps {
  isOpen: boolean
  onClose: () => void
}

type TabType = 'magnet' | 'url' | 'file'

export function AddTorrentModal({ isOpen, onClose }: AddTorrentModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>('magnet')
  const [magnetUri, setMagnetUri] = useState('')
  const [torrentUrl, setTorrentUrl] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()

  const addMagnetMutation = useMutation({
    mutationFn: (uri: string) => torrentsApi.addMagnet(uri),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['torrents'] })
      toast.success('Torrent added successfully')
      handleClose()
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to add torrent')
    },
  })

  const addUrlMutation = useMutation({
    mutationFn: (url: string) => torrentsApi.addUrl(url),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['torrents'] })
      toast.success('Torrent added successfully')
      handleClose()
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to add torrent')
    },
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) => torrentsApi.upload(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['torrents'] })
      toast.success('Torrent uploaded successfully')
      handleClose()
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to upload torrent')
    },
  })

  const handleClose = () => {
    setMagnetUri('')
    setTorrentUrl('')
    setSelectedFile(null)
    onClose()
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    if (activeTab === 'magnet' && magnetUri) {
      addMagnetMutation.mutate(magnetUri)
    } else if (activeTab === 'url' && torrentUrl) {
      addUrlMutation.mutate(torrentUrl)
    } else if (activeTab === 'file' && selectedFile) {
      uploadMutation.mutate(selectedFile)
    }
  }

  const isLoading = addMagnetMutation.isPending || addUrlMutation.isPending || uploadMutation.isPending

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={handleClose} />
      
      <div className="relative bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Add Torrent</h2>
          <button
            onClick={handleClose}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-200">
          {[
            { id: 'magnet' as TabType, label: 'Magnet Link', icon: LinkIcon },
            { id: 'url' as TabType, label: 'Torrent URL', icon: LinkIcon },
            { id: 'file' as TabType, label: 'Upload File', icon: Upload },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
                activeTab === tab.id
                  ? 'border-primary-600 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              )}
            >
              <tab.icon className="w-4 h-4" />
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <form onSubmit={handleSubmit} className="p-6">
          {activeTab === 'magnet' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Magnet URI
              </label>
              <textarea
                value={magnetUri}
                onChange={(e) => setMagnetUri(e.target.value)}
                placeholder="magnet:?xt=urn:btih:..."
                className="input min-h-[100px] resize-none"
                autoFocus
              />
              <p className="mt-2 text-sm text-gray-500">
                Paste a magnet link starting with magnet:?
              </p>
            </div>
          )}

          {activeTab === 'url' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Torrent URL
              </label>
              <input
                type="url"
                value={torrentUrl}
                onChange={(e) => setTorrentUrl(e.target.value)}
                placeholder="https://example.com/file.torrent"
                className="input"
                autoFocus
              />
              <p className="mt-2 text-sm text-gray-500">
                Enter a direct link to a .torrent file
              </p>
            </div>
          )}

          {activeTab === 'file' && (
            <div>
              <input
                ref={fileInputRef}
                type="file"
                accept=".torrent"
                onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
                className="hidden"
              />
              <div
                onClick={() => fileInputRef.current?.click()}
                className={cn(
                  'border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors',
                  selectedFile
                    ? 'border-primary-300 bg-primary-50'
                    : 'border-gray-300 hover:border-gray-400'
                )}
              >
                <Upload className="w-10 h-10 mx-auto text-gray-400 mb-3" />
                {selectedFile ? (
                  <p className="text-sm text-primary-600 font-medium">
                    {selectedFile.name}
                  </p>
                ) : (
                  <>
                    <p className="text-sm text-gray-600 font-medium">
                      Click to upload a .torrent file
                    </p>
                    <p className="text-xs text-gray-500 mt-1">
                      or drag and drop
                    </p>
                  </>
                )}
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-3 mt-6">
            <button
              type="button"
              onClick={handleClose}
              className="btn-secondary"
              disabled={isLoading}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn-primary"
              disabled={
                isLoading ||
                (activeTab === 'magnet' && !magnetUri) ||
                (activeTab === 'url' && !torrentUrl) ||
                (activeTab === 'file' && !selectedFile)
              }
            >
              {isLoading ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Adding...
                </>
              ) : (
                'Add Torrent'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
