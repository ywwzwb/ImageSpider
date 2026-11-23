import test from 'node:test'
import assert from 'node:assert/strict'
import { computeRangeIds, rectsIntersect, unionSelection } from './selection.ts'

const ids = ['a', 'b', 'c', 'd', 'e']

test('computeRangeIds 正向包含两端且按页面顺序', () => {
  assert.deepEqual(computeRangeIds(ids, 'b', 'd'), ['b', 'c', 'd'])
})

test('computeRangeIds 反向也按页面顺序', () => {
  assert.deepEqual(computeRangeIds(ids, 'd', 'b'), ['b', 'c', 'd'])
})

test('computeRangeIds 同一张返回自身', () => {
  assert.deepEqual(computeRangeIds(ids, 'c', 'c'), ['c'])
})

test('computeRangeIds 锚点或目标不存在返回空数组', () => {
  assert.deepEqual(computeRangeIds(ids, 'x', 'c'), [])
  assert.deepEqual(computeRangeIds(ids, 'a', 'x'), [])
})

test('rectsIntersect 有重叠返回 true', () => {
  assert.equal(
    rectsIntersect({ left: 0, right: 10, top: 0, bottom: 10 }, { left: 5, right: 15, top: 5, bottom: 15 }),
    true
  )
})

test('rectsIntersect 完全分离或仅边缘相切返回 false', () => {
  assert.equal(
    rectsIntersect({ left: 0, right: 10, top: 0, bottom: 10 }, { left: 20, right: 30, top: 20, bottom: 30 }),
    false
  )
  assert.equal(
    rectsIntersect({ left: 0, right: 10, top: 0, bottom: 10 }, { left: 10, right: 20, top: 0, bottom: 10 }),
    false
  )
})

test('unionSelection 保留原有选择并加入命中项，不修改原集合', () => {
  const initial = new Set(['a'])
  assert.deepEqual(unionSelection(initial, ['c', 'b']), ['a', 'c', 'b'])
  assert.deepEqual(Array.from(initial), ['a'])
})
