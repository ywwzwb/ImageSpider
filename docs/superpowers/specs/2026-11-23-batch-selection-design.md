# 批量选择：Shift 连选与鼠标拖拽框选

## 背景与目标

批量模式下目前只能逐个点击卡片或复选框，选几十张图很费劲。

目标：

- 按住 Shift 点击，自动把锚点到当前点击之间的图片整段勾选/取消。
- 在批量模式下按鼠标左键拖动，拉出选框，框到的图片一起选中。
- 框选采用**叠加**语义：在已有选择上继续加选，不清空之前的选择。

## 非目标

- 不做触屏框选（仅鼠标；移动端保持现状）。
- 不跨页保留选择（沿用现有逻辑：翻页/筛选后清理不在当前页的选中项）。
- 不引入第三方框选库。
- 不改变批量删除、批量重新下载、批量标记状态的接口。

## 交互设计

### Shift 点击

- 锚点 = 最近一次普通点击（卡片或复选框）的图片。
- 普通点击：切换单张选中状态，并更新锚点。
- Shift + 点击：把锚点到当前点击之间（按当前页显示顺序，含两端）整段设为“当前这张的相反状态”：
  - 当前这张已选中 → 整段取消选中；
  - 当前这张未选中 → 整段选中。
  - 选区之外的已选项不受影响。
- Shift 点击不移动锚点，便于继续拉长/缩短范围。
- 没有锚点、或锚点不在当前页时，Shift 点击退化为普通点击。
- 翻页、修改筛选、重新加载列表时清空锚点。

### 拖拽框选

- 仅在批量模式下生效。
- 在卡片或网格空白处按下鼠标左键都可以开始拖拽；交互控件（按钮、复选框、标签、分页器、批量操作栏）上不触发。
- 鼠标移动超过 5px 才开始框选，避免和普通单击冲突。
- 拖拽时显示半透明蓝色选框，被框中的卡片实时高亮。
- 松手生效，框选结果叠加到已有选择。
- 拖拽结束后抑制本次误触产生的 click，避免把刚开始拖拽的那张卡片切换掉。
- 拖拽期间禁用页面文本选择和图片原生拖拽。
- 窗口缩放/滚动时以鼠标移动时的最新坐标为准（不额外实现自动滚动）。

## 技术实现

### 组件职责

- `ImageCard.vue`
  - 卡片点击和复选框点击都改为向父组件抛出 `batch-select` 事件，携带 `{ id, shiftKey, ctrlKey/metaKey }`；是否预览仍由卡片根据 `batchMode` 决定。
  - 图片设置 `draggable="false"`，阻止原生拖拽。
- `ImageGrid.vue`
  - 维护 `anchorId`；处理 `batch-select`：Shift 时计算范围并批量设置，否则切换单张并更新锚点。
  - 处理 `mousedown/mousemove/mouseup`：维护框选状态、选框位置、拖拽前的选择快照、是否叠加。
  - 给每个卡片列加 `data-image-id`，通过 `getBoundingClientRect()` 判断与选框是否相交。
  - 选框用 `Teleport` 渲染到 `body`，`position: fixed`、`pointer-events: none`。
  - 卸载时移除 document 上的监听。
- `stores/app.ts`
  - 增加 `setSelectedImages(ids)`（整体替换）和 `addSelectedImages(ids)`（并集），供框选实时更新。
  - 现有 `toggleImageSelection / selectAll / clearSelection / getSelectedIds` 保持不变。

### 纯函数

抽到 `frontend/src/utils/selection.ts`，便于单元测试：

- `computeRangeIds(ids, anchorId, targetId): string[]`：返回两端之间的 id 列表；任一 id 不存在时返回空数组。
- `rectsIntersect(a, b): boolean`。

### 样式

- `.image-grid` 拖拽中加 `user-select: none`。
- 选框：`border: 1px solid #1890ff`，`background: rgba(24, 144, 255, 0.12)`，`z-index: 1000`。

## 测试与验证

- 纯函数用 Node 内置 `node:test` 测试（Node 24 原生支持 .ts）：覆盖正向/反向范围、边界、id 不存在、矩形相交/相离/相切。
- `npm run build` 通过（`vue-tsc` 类型检查 + Vite 构建）。
- 手工验证：批量模式下 Shift 连选/反选、拖拽框选、叠加、单击不误触、翻页后锚点重置、非批量模式行为不变。
