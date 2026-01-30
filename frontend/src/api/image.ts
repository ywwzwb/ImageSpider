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

    return client.get(`/${sourceId}/images`, { params }).then(res => res.data)
  },

  getImage(sourceId: string, id: string): Promise<ImageMeta> {
    return client.get(`/${sourceId}/image/${id}`).then(res => res.data)
  },

  batchDelete(sourceId: string, ids: string[]): Promise<BatchOperationResponse> {
    const data: BatchOperationRequest = { ids }
    return client.delete(`/${sourceId}/images`, { data }).then(res => res.data)
  },

  batchRedownload(sourceId: string, ids: string[]): Promise<BatchOperationResponse> {
    const data: BatchOperationRequest = { ids }
    return client.post(`/${sourceId}/images/redownload`, data).then(res => res.data)
  }
}
