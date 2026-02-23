import client from './client'
import type { ImageList, ImageMeta, BatchOperationRequest, BatchOperationResponse } from '@/types/api'

export const imageApi = {
  getImages(
    sourceId: string,
    options: {
      tags?: string[]
      integrityStatus?: number[]
      offset?: number
      limit?: number
    } = {}
  ): Promise<ImageList> {
    const params: Record<string, any> = {
      offset: options.offset || 0,
      limit: options.limit || 50
    }

    if (options.tags && options.tags.length > 0) {
      params.tag = options.tags
    }

    if (options.integrityStatus && options.integrityStatus.length > 0) {
      params.integrity_status = options.integrityStatus
    }

    // Use paramsSerializer to send array as repeated params (tag=val1&tag=val2)
    // which Gin can correctly parse
    return client.get(`/${sourceId}/images`, {
      params,
      paramsSerializer: (params) => {
        const parts: string[] = []
        Object.entries(params).forEach(([key, value]) => {
          if (Array.isArray(value)) {
            value.forEach((v) => {
              parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(v)}`)
            })
          } else {
            parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
          }
        })
        return parts.join('&')
      }
    })
  },

  getImage(sourceId: string, id: string): Promise<ImageMeta> {
    return client.get(`/${sourceId}/image/${id}`)
  },

  batchDelete(sourceId: string, ids: string[]): Promise<BatchOperationResponse> {
    const data: BatchOperationRequest = { ids }
    return client.delete(`/${sourceId}/images`, { data })
  },

  batchRedownload(sourceId: string, ids: string[]): Promise<BatchOperationResponse> {
    const data: BatchOperationRequest = { ids }
    return client.post(`/${sourceId}/images/redownload`, data)
  }
}
