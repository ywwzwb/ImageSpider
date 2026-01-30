# ImageSpider Frontend

基于 Vue 3 + TypeScript + Ant Design Vue 的现代化图片管理界面。

## 技术栈

- Vue 3.5 (Composition API)
- TypeScript 5
- Vite 5
- Ant Design Vue 4
- Pinia 2 (状态管理)
- Axios (HTTP 客户端)

## 功能特性

### 核心功能
- **多源管理**: 支持切换不同的图片源 (source)
- **标签筛选**: 按标签筛选图片，支持多选
- **完整性筛选**: 按图片完整性状态筛选（正常/破损/未知）
- **图片浏览**: 网格布局展示图片缩略图
- **图片预览**: 全屏预览大图，支持键盘导航
- **批量操作**: 批量选择和操作图片

### 图片操作
- 删除单张/批量图片
- 重新下载单张/批量图片
- 设置标签封面
- 通过标签快速筛选

### 用户体验
- 响应式设计，支持移动端
- 键盘快捷键支持
- 加载状态提示
- 操作确认对话框
- 错误消息提示

## 快捷键

- `←` / `→`: 在预览模式中切换图片
- `Delete`: 删除当前选中的图片
- `Ctrl+A`: 全选当前页的图片
- `ESC`: 关闭预览或取消操作

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览构建结果
npm run preview
```

开发服务器将运行在 http://localhost:5173

API 请求会自动代理到 http://localhost:8080

## 构建输出

构建产物输出到 `/embed/www` 目录，由 Go 后端通过 embed 提供服务。

## API 接口

前端使用以下后端 API：

- `GET /api/sources` - 获取所有可用的 source_id
- `GET /api/:sourceid/tags` - 获取标签列表
- `GET /api/:sourceid/images` - 获取图片列表（支持筛选）
- `GET /api/:sourceid/image/:id` - 获取单张图片详情
- `DELETE /api/:sourceid/images` - 批量删除图片
- `POST /api/:sourceid/images/redownload` - 批量重新下载
- `POST /api/:sourceid/tags/:tag/cover` - 设置标签封面
- `GET /image/*` - 获取图片文件

## 项目结构

```
frontend/
├── src/
│   ├── api/              # API 调用封装
│   │   ├── client.ts     # Axios 实例
│   │   ├── source.ts     # 源相关 API
│   │   ├── tag.ts        # 标签相关 API
│   │   └── image.ts      # 图片相关 API
│   ├── components/       # Vue 组件
│   │   ├── IntegrityFilter.vue   # 完整性筛选
│   │   ├── TagFilter.vue         # 标签筛选
│   │   ├── ImageCard.vue         # 单张图片卡片
│   │   ├── ImageGrid.vue         # 图片网格容器
│   │   └── ImagePreview.vue      # 图片预览模态框
│   ├── composables/      # 组合式函数
│   │   └── useKeyboardShortcuts.ts  # 键盘快捷键
│   ├── stores/           # Pinia 状态管理
│   │   ├── app.ts        # 全局状态
│   │   ├── source.ts     # 源状态
│   │   └── filter.ts     # 筛选状态
│   ├── types/            # TypeScript 类型
│   │   └── api.ts        # API 相关类型
│   ├── utils/            # 工具函数
│   │   └── format.ts     # 格式化函数
│   ├── App.vue           # 根组件
│   └── main.ts           # 入口文件
├── index.html            # HTML 模板
├── package.json          # 依赖管理
├── vite.config.ts        # Vite 配置
├── tsconfig.json         # TypeScript 配置
└── tsconfig.node.json    # Node 环境 TS 配置
```

## 性能优化建议

当前构建输出单个较大的 JS 文件，可以通过以下方式优化：

1. **代码分割**: 使用动态导入拆分路由和组件
2. **图片懒加载**: 实现虚拟滚动优化大量图片列表
3. **缓存策略**: 为 API 响应添加缓存
4. **CDN**: 将静态资源部署到 CDN

## 浏览器支持

- Chrome >= 60
- Firefox >= 55
- Safari >= 11
- Edge >= 79
