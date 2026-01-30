// Image integrity status enum
export enum ImageIntegrityStatus {
  UNKNOWN = 0,
  GOOD = 1,
  BAD = -1
}

// Tag types
export interface TagInfo {
  tag: string
  count: number
  cover: ImageMeta
}

export interface TagList {
  tagList: TagInfo[]
  totalCount: number
}

// Image types
export interface ImageMeta {
  id: string
  tags: string[]
  localPath: string | null
  imageURL: string
  postTime: string
  sourceId: string
  integrityStatus: ImageIntegrityStatus
}

export interface ImageList {
  imageList: ImageMeta[]
  totalCount: number
}

// API response types
export interface SourcesResponse {
  sources: string[]
}

export interface BatchOperationResponse {
  deleted?: number
  redownloaded?: number
  total: number
}

export interface SetTagCoverRequest {
  imageId: string
}

export interface BatchOperationRequest {
  ids: string[]
}
