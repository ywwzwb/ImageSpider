import { ImageIntegrityStatus } from '@/types/api'

export function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

export function getIntegrityStatusText(status: ImageIntegrityStatus): string {
  switch (status) {
    case ImageIntegrityStatus.UNKNOWN:
      return '未知'
    case ImageIntegrityStatus.GOOD:
      return '正常'
    case ImageIntegrityStatus.BAD:
      return '破损'
    default:
      return '未知'
  }
}

export function getIntegrityStatusColor(status: ImageIntegrityStatus): string {
  switch (status) {
    case ImageIntegrityStatus.UNKNOWN:
      return 'default'
    case ImageIntegrityStatus.GOOD:
      return 'success'
    case ImageIntegrityStatus.BAD:
      return 'error'
    default:
      return 'default'
  }
}

export function getThumbnailPath(imagePath: string | null, size = 320): string | null {
  if (!imagePath) return null

  // Convert /image/ prefix to relative path if needed
  const cleanPath = imagePath.replace(/^\/image\//, '')

  // Insert thumbnail size before file extension
  const lastDotIndex = cleanPath.lastIndexOf('.')
  if (lastDotIndex === -1) {
    return `/image/${cleanPath}@${size}`
  }

  const name = cleanPath.substring(0, lastDotIndex)
  const ext = cleanPath.substring(lastDotIndex)
  return `/image/${name}@${size}${ext}`
}

export function truncateText(text: string, maxLength = 50): string {
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

export function classNames(...classes: (string | false | null | undefined)[]): string {
  return classes.filter(Boolean).join(' ')
}
