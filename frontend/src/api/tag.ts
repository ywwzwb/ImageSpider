import client from './client'
import type { TagList, SetTagCoverRequest } from '@/types/api'

export const tagApi = {
  getTags(sourceId: string, offset = 0, limit = 50): Promise<TagList> {
    return client.get(`/${sourceId}/tags`, {
      params: { offset, limit }
    })
  },

  setTagCover(sourceId: string, tag: string, imageId: string): Promise<void> {
    const data: SetTagCoverRequest = { imageId }
    return client.post(`/${sourceId}/tags/${tag}/cover`, data)
  }
}
