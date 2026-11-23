# 批量选择 Shift 连选与框选 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 批量模式下支持 Shift 范围勾选/反选和鼠标拖拽框选（叠加）。

**Architecture:** 把范围计算和矩形相交抽成纯函数并用 `node:test` 测试；`ImageCard` 只上报带修饰键的选择事件；`ImageGrid` 维护锚点、拖拽状态和选框；Pinia store 增加整体设置/并集方法。

**Tech Stack:** Vue 3 + TypeScript + Pinia + Ant Design Vue 4；Node 24 内置 `node:test`（原生执行 `.ts`）。

**Spec:** `docs/superpowers/specs/2026-11-23-batch-selection-design.md`

## Global Constraints

- 只改 `frontend/`，不引入运行时第三方依赖；测试仅用 Node 内置模块。
- 框选必须是**叠加**语义，不清空已有选择。
- Shift 点击按当前点击图片的相反状态处理整段（已选→取消，未选→选中）。
- 拖拽阈值 5px；仅批量模式、仅鼠标左键。
- 不改变现有批量删除/重新下载/标记状态的接口与行为。
- 非批量模式行为保持不变（点击仍是预览）。

## Review Focus

- 拖拽结束后的 click 不能把起始卡片误切换（review：Task 5 在 grid 捕获阶段吞掉一次 click）。
- 没有锚点或锚点已被翻页清掉时，Shift 点击不能报错（review：Task 4 退化为普通点击）。
- 在标签、复选框、按钮、分页器上按下鼠标不能触发框选（review：Task 5 的 `closest` 过滤）。
- 选框用视口坐标 + `position: fixed`，页面滚动后位置仍正确（review：Task 5 使用 `clientX/clientY`）。
- 退出批量模式、翻页、换筛选后锚点必须失效（review：Task 4 的重置逻辑）。

---

### Task 1: 纯函数 `computeRangeIds` / `rectsIntersect`

**Files:**
- Create: `frontend/src/utils/selection.ts`
- Create: `frontend/src/utils/selection.test.ts`
- Modify: `frontend/package.json`（加 `test` 脚本）

**Interfaces:**
- Consumes: 无
- Produces:
  - `interface SelectionRect { left: number; right: number; top: number; bottom: number }`
  - `computeRangeIds(ids: string[], anchorId: string, targetId: string): string[]`
  - `rectsIntersect(a: SelectionRect, b: SelectionRect): boolean`

- [ ] **Step 1: 写失败测试**

`frontend/src/utils/selection.test.ts`:

```ts
import test from 'node:test'
import assert from 'node:assert/strict'
import { computeRangeIds, rectsIntersect } from './selection.ts'

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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && node --test src/utils/selection.test.ts`
Expected: FAIL，报找不到 `./selection.ts`

- [ ] **Step 3: 实现纯函数**

`frontend/src/utils/selection.ts`：`computeRangeIds` 先 `indexOf` 两个 id，任一为 -1 返回 `[]`，否则取 `slice(min, max + 1)`；`rectsIntersect` 用严格不等号判断四个方向都有重叠。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && node --test src/utils/selection.test.ts`
Expected: 6 tests PASS

- [ ] **Step 5: 加 npm 脚本并提交**

`package.json` 的 `scripts` 加 `"test": "node --test src/utils/*.test.ts"`。

```bash
git add frontend/src/utils/selection.ts frontend/src/utils/selection.test.ts frontend/package.json
git commit -m "feat(frontend): 添加批量选择范围计算工具"
```

---

### Task 2: Pinia store 批量设置方法

**Files:**
- Modify: `frontend/src/stores/app.ts`

**Interfaces:**
- Consumes: 无
- Produces:
  - `setSelectedImages(ids: string[]): void`（整体替换）
  - `addSelectedImages(ids: string[]): void`（并集）

- [ ] **Step 1: 实现两个方法**

在 `clearSelection` 附近加入：

```ts
function setSelectedImages(ids: string[]) {
  selectedImages.value = new Set(ids)
}

function addSelectedImages(ids: string[]) {
  ids.forEach(id => selectedImages.value.add(id))
}
```

并加入 return 对象。

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add frontend/src/stores/app.ts
git commit -m "feat(frontend): store 支持批量设置选择"
```

---

### Task 3: ImageCard 上报带修饰键的选择事件

**Files:**
- Modify: `frontend/src/components/ImageCard.vue`

**Interfaces:**
- Consumes: store 现有方法
- Produces: 新事件 `batch-select`，参数 `{ id: string; shiftKey: boolean; ctrlKey: boolean }`

- [ ] **Step 1: 加事件类型和处理函数**

`Emits` 增加：

```ts
(e: 'batch-select', payload: { id: string; shiftKey: boolean; ctrlKey: boolean }): void
```

`handleImageClick(event: MouseEvent)`：批量模式改为 `emit('batch-select', { id: props.image.id, shiftKey: event.shiftKey, ctrlKey: event.ctrlKey || event.metaKey })`，否则仍 `emit('preview', props.image)`。

新增 `handleSelectionClick(event: MouseEvent)`，同样 emit `batch-select`。

- [ ] **Step 2: 改模板**

- 卡片点击：`@click="handleImageClick"`（Vue 自动传事件）。
- 复选框：外层 div 加 `@click.stop="handleSelectionClick"`，`a-checkbox` 只保留 `:checked`，删除 `@click.stop` 和 `@change`，避免双重切换。
- 图片加 `draggable="false"`。
- 删除不再使用的 `handleSelectionChange`。

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add frontend/src/components/ImageCard.vue
git commit -m "feat(frontend): 图片卡片上报 Shift 选择事件"
```

---

### Task 4: ImageGrid 实现 Shift 范围选择

**Files:**
- Modify: `frontend/src/components/ImageGrid.vue`

**Interfaces:**
- Consumes: `computeRangeIds`（Task 1）、`batch-select` 事件（Task 3）、`setSelectedImages`（Task 2）
- Produces: `anchorId`、`handleBatchSelect(payload)`

- [ ] **Step 1: 接入事件与锚点**

- 模板 `ImageCard` 加 `@batch-select="handleBatchSelect"`。
- 新增 `const anchorId = ref<string | null>(null)`。
- 实现：

```ts
function handleBatchSelect(payload: { id: string; shiftKey: boolean; ctrlKey: boolean }) {
  if (payload.shiftKey && anchorId.value) {
    const range = computeRangeIds(currentImageIds.value, anchorId.value, payload.id)
    if (range.length > 0) {
      const shouldSelect = !appStore.isImageSelected(payload.id)
      const next = new Set(appStore.selectedImages)
      range.forEach(id => (shouldSelect ? next.add(id) : next.delete(id)))
      appStore.setSelectedImages(Array.from(next))
      return
    }
  }
  appStore.toggleImageSelection(payload.id)
  anchorId.value = payload.id
}
```

- [ ] **Step 2: 重置锚点**

- `loadImages()` 开始时 `anchorId.value = null`。
- `watch(() => appStore.batchMode, on => { if (!on) anchorId.value = null })`。
- 「取消选择」按钮改为 `handleClearSelection`：清空选择并把锚点置空。

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add frontend/src/components/ImageGrid.vue
git commit -m "feat(frontend): 支持 Shift 范围勾选反选"
```

---

### Task 5: ImageGrid 鼠标拖拽框选

**Files:**
- Modify: `frontend/src/components/ImageGrid.vue`

**Interfaces:**
- Consumes: `rectsIntersect`（Task 1）、`setSelectedImages`（Task 2）
- Produces: 无对外接口

- [ ] **Step 1: 模板改造**

- 根 div 加 `ref="gridRef"`、`@mousedown="handleGridMouseDown"`、`@click.capture="handleGridClickCapture"`、`:class="{ 'is-dragging': marquee.active }"`。
- 每个 `a-col` 加 `:data-image-id="image.id"`。
- 末尾加：

```vue
<Teleport to="body">
  <div v-if="marquee.active" class="marquee-box" :style="marqueeStyle" />
</Teleport>
```

- [ ] **Step 2: 实现拖拽逻辑**

```ts
const gridRef = ref<HTMLElement>()
const marquee = reactive({ active: false, startX: 0, startY: 0, currentX: 0, currentY: 0 })
let pendingStart: { x: number; y: number; additive: boolean; initial: Set<string> } | null = null
let dragStarted = false
let suppressNextClick = false
let marqueeLastId: string | null = null
let rafId = 0

const marqueeStyle = computed(() => ({
  left: `${Math.min(marquee.startX, marquee.currentX)}px`,
  top: `${Math.min(marquee.startY, marquee.currentY)}px`,
  width: `${Math.abs(marquee.currentX - marquee.startX)}px`,
  height: `${Math.abs(marquee.currentY - marquee.startY)}px`
}))
```

- `handleGridMouseDown(event)`：非批量模式、非左键、或 `event.target.closest('button, a, input, .ant-checkbox-wrapper, .ant-tag, .ant-pagination, .batch-bar')` 命中则返回；记录起点、`additive = shift/ctrl/meta`、初始选择快照；挂 `window` 的 mousemove/mouseup。
- `handleGridMouseMove(event)`：未开始且位移 < 5px 返回；首次开始时置 `marquee.active = true`、`document.body.style.userSelect = 'none'`、清空浏览器选区；更新 `currentX/currentY` 并用 `requestAnimationFrame` 调 `applyMarqueeSelection()`。
- `applyMarqueeSelection()`：用 `gridRef.value.querySelectorAll<HTMLElement>('[data-image-id]')`，`rectsIntersect` 命中则加入 `next`；`next` 初始为 `additive ? 初始快照 : 空集`；最后 `appStore.setSelectedImages(Array.from(next))`，并记录命中的最后一个 id 到 `marqueeLastId`。
- `handleGridMouseUp()`：移除 window 监听；若 `dragStarted`，置 `suppressNextClick = true`、`marqueeLastId` 存在时更新锚点、用 `setTimeout` 100ms 后复位 `suppressNextClick`；清理 `marquee.active`、`pendingStart`、`dragStarted`、body userSelect。
- `handleGridClickCapture(event)`：`suppressNextClick` 为真时 `event.stopPropagation()`、`event.preventDefault()` 并复位。
- `onBeforeUnmount` 移除监听并取消 rAF/定时器。

- [ ] **Step 3: 样式**

```css
.image-grid.is-dragging {
  user-select: none;
}

.marquee-box {
  position: fixed;
  border: 1px solid #1890ff;
  background: rgba(24, 144, 255, 0.12);
  pointer-events: none;
  z-index: 1000;
}
```

（`.marquee-box` 在 Teleport 到 body 后仍受 scoped 样式影响，需要保留 scoped；如不生效则加 `:deep` 或改用全局样式。）

- [ ] **Step 4: 类型检查 + 构建**

Run: `cd frontend && npm run build`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/ImageGrid.vue
git commit -m "feat(frontend): 支持鼠标拖拽框选图片"
```

---

### Task 6: 全量验证

- [ ] **Step 1: 跑单测**

Run: `cd frontend && npm test`
Expected: 6 tests PASS

- [ ] **Step 2: 跑构建**

Run: `cd frontend && npm run build`
Expected: PASS

- [ ] **Step 3: 手工验收清单**

在浏览器里核对：

1. 批量模式：点 A，Shift 点 D → A–D 选中；再 Shift 点 B → 只剩 B–D 选中（反向取消）。
2. 拖框经过已选和未选卡片 → 只加选，不清空之前的。
3. 从空白处和卡片上都能拖框；从标签/复选框/按钮/分页器上按下不触发框选。
4. 拖框结束后起始卡片不会误切换。
5. 翻页、换筛选后 Shift 点击退化为普通点击。
6. 退出批量模式后点击仍是预览。

- [ ] **Step 4: 提交验收记录或修正**

如手工验收发现问题，修复后补提交。
