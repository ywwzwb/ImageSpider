export interface SelectionRect {
  left: number
  right: number
  top: number
  bottom: number
}

export function computeRangeIds(ids: string[], anchorId: string, targetId: string): string[] {
  const anchorIndex = ids.indexOf(anchorId)
  const targetIndex = ids.indexOf(targetId)
  if (anchorIndex === -1 || targetIndex === -1) {
    return []
  }
  const start = Math.min(anchorIndex, targetIndex)
  const end = Math.max(anchorIndex, targetIndex)
  return ids.slice(start, end + 1)
}

export function rectsIntersect(a: SelectionRect, b: SelectionRect): boolean {
  return a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top
}
