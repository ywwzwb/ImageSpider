import client from './client'
import type { SourcesResponse } from '@/types/api'

export const sourceApi = {
  getSources(): Promise<SourcesResponse> {
    return client.get('/sources')
  }
}
