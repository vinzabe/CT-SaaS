import { useState } from 'react'
import { 
  Pause, 
  Play, 
  Trash2, 
  Download, 
  ChevronDown, 
  ChevronUp,
  File,
  Users,
  Upload,
  Clock
} from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import toast from 'react-hot-toast'
import { torrentsApi } from '../lib/api'
import { 
  cn, 
  formatBytes, 
  formatSpeed, 
  formatProgress,
  getStatusColor,
  estimateTimeRemaining
} from '../lib/utils'
import type { Torrent } from '../types'

interface TorrentCardProps {
  torrent: Torrent
}

export function TorrentCard({ torrent }: TorrentCardProps) {
  const [expanded, setExpanded] = useState(false)
  const queryClient = useQueryClient()

  const pauseMutation = useMutation({
    mutationFn: () => torrentsApi.pause(torrent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['torrents'] })
      toast.success('Torrent paused')
    },
    onError: () => toast.error('Failed to pause torrent'),
  })

  const resumeMutation = useMutation({
    mutationFn: () => torrentsApi.resume(torrent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['torrents'] })
      toast.success('Torrent resumed')
    },
    onError: () => toast.error('Failed to resume torrent'),
  })

  const deleteMutation = useMutation({
    mutationFn: () => torrentsApi.delete(torrent.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['torrents'] })
      toast.success('Torrent deleted')
    },
    onError: () => toast.error('Failed to delete torrent'),
  })

  const downloadMutation = useMutation({
    mutationFn: ({ filePath, useZip }: { filePath: string; useZip: boolean }) => 
      torrentsApi.createDownloadToken(torrent.id, filePath, useZip),
    onSuccess: (data) => {
      window.open(data.download_url, '_blank')
    },
    onError: () => toast.error('Failed to generate download link'),
  })

  const hasMultipleFiles = torrent.files && torrent.files.length > 1
  const hasZip = !!torrent.zip_path

  const isDownloading = torrent.status === 'downloading'
  const isCompleted = torrent.status === 'completed' || torrent.status === 'seeding'
  const isPaused = torrent.status === 'paused'

  return (
    <div className="card overflow-hidden">
      <div className="p-4">
        <div className="flex items-start gap-4">
          {/* Icon */}
          <div className={cn(
            'w-12 h-12 rounded-lg flex items-center justify-center flex-shrink-0',
            isCompleted ? 'bg-green-100' : 'bg-primary-100'
          )}>
            <Download className={cn(
              'w-6 h-6',
              isCompleted ? 'text-green-600' : 'text-primary-600'
            )} />
          </div>

          {/* Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="text-sm font-medium text-gray-900 truncate">
                {torrent.name || 'Loading metadata...'}
              </h3>
              <span className={cn(
                'px-2 py-0.5 text-xs font-medium rounded-full capitalize',
                getStatusColor(torrent.status)
              )}>
                {torrent.status}
              </span>
            </div>

            {/* Stats row */}
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500">
              <span>{formatBytes(torrent.total_size)}</span>
              
              {isDownloading && (
                <>
                  <span className="flex items-center gap-1">
                    <Download className="w-3.5 h-3.5" />
                    {formatSpeed(torrent.download_speed)}
                  </span>
                  <span className="flex items-center gap-1">
                    <Upload className="w-3.5 h-3.5" />
                    {formatSpeed(torrent.upload_speed)}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5" />
                    {estimateTimeRemaining(torrent.total_size, torrent.downloaded_size, torrent.download_speed)}
                  </span>
                </>
              )}
              
              <span className="flex items-center gap-1">
                <Users className="w-3.5 h-3.5" />
                {torrent.seeds} seeds, {torrent.peers} peers
              </span>
            </div>

            {/* Progress bar */}
            <div className="mt-3">
              <div className="flex justify-between text-xs mb-1">
                <span className="text-gray-500">
                  {formatBytes(torrent.downloaded_size)} of {formatBytes(torrent.total_size)}
                </span>
                <span className="font-medium text-gray-900">
                  {formatProgress(torrent.progress)}
                </span>
              </div>
              <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                <div
                  className={cn(
                    'h-full rounded-full transition-all duration-300',
                    isDownloading ? 'bg-primary-600 progress-striped' : 'bg-green-600'
                  )}
                  style={{ width: `${Math.min(100, torrent.progress)}%` }}
                />
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-1">
            {isDownloading && (
              <button
                onClick={() => pauseMutation.mutate()}
                disabled={pauseMutation.isPending}
                className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
                title="Pause"
              >
                <Pause className="w-5 h-5" />
              </button>
            )}
            
            {isPaused && (
              <button
                onClick={() => resumeMutation.mutate()}
                disabled={resumeMutation.isPending}
                className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
                title="Resume"
              >
                <Play className="w-5 h-5" />
              </button>
            )}

            {isCompleted && (
              <button
                onClick={() => {
                  if (hasMultipleFiles && hasZip) {
                    // Download zip for multi-file torrents
                    downloadMutation.mutate({ filePath: torrent.zip_path!, useZip: true })
                  } else if (torrent.files && torrent.files.length > 0) {
                    downloadMutation.mutate({ filePath: torrent.files[0].path, useZip: false })
                  } else if (torrent.name) {
                    downloadMutation.mutate({ filePath: torrent.name, useZip: false })
                  }
                }}
                disabled={downloadMutation.isPending || (!torrent.files?.length && !torrent.name)}
                className="p-2 text-primary-600 hover:text-primary-700 hover:bg-primary-50 rounded-lg"
                title={hasMultipleFiles ? "Download ZIP" : "Download"}
              >
                <Download className="w-5 h-5" />
              </button>
            )}

            <button
              onClick={() => deleteMutation.mutate()}
              disabled={deleteMutation.isPending}
              className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg"
              title="Delete"
            >
              <Trash2 className="w-5 h-5" />
            </button>

            {torrent.files && torrent.files.length > 0 && (
              <button
                onClick={() => setExpanded(!expanded)}
                className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
              >
                {expanded ? (
                  <ChevronUp className="w-5 h-5" />
                ) : (
                  <ChevronDown className="w-5 h-5" />
                )}
              </button>
            )}
          </div>
        </div>

        {/* Error message */}
        {torrent.error_message && (
          <div className="mt-3 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
            {torrent.error_message}
          </div>
        )}
      </div>

      {/* Files list */}
      {expanded && torrent.files && torrent.files.length > 0 && (
        <div className="border-t border-gray-200 bg-gray-50 px-4 py-3">
          <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
            Files ({torrent.files.length})
          </h4>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {torrent.files.map((file, index) => (
              <div
                key={index}
                className="flex items-center gap-3 p-2 bg-white rounded-lg border border-gray-200"
              >
                <File className="w-4 h-4 text-gray-400 flex-shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-900 truncate">{file.path}</p>
                  <p className="text-xs text-gray-500">
                    {formatBytes(file.size)} - {formatProgress(file.progress)}
                  </p>
                </div>
                {isCompleted && (
                  <button
                    onClick={() => downloadMutation.mutate({ filePath: file.path, useZip: false })}
                    disabled={downloadMutation.isPending}
                    className="p-1.5 text-primary-600 hover:bg-primary-50 rounded"
                    title="Download"
                  >
                    <Download className="w-4 h-4" />
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
