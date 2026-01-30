# 编译指南

## 快速编译

在项目根目录执行：

```bash
./build.sh        # 编译前端和后端（默认）
./build.sh all    # 编译前端和后端
./build.sh frontend    # 只编译前端
./build.sh backend     # 只编译后端
./build.sh clean       # 清理编译产物
./build.sh help        # 显示帮助信息
```

## Docker 编译

### 构建镜像

```bash
docker build -t imagespider:latest .
```

### 运行容器

```bash
docker run -d \
  --name imagespider \
  -p 8080:8080 \
  -v /path/to/config.yaml:/config/config.yaml:ro \
  -v /path/to/images:/app/images \
  imagespider:latest
```

参数说明：
- `-p 8080:8080` - 端口映射
- `-v /path/to/config.yaml:/config/config.yaml:ro` - 配置文件（只读）
- `-v /path/to/images:/app/images` - 图片存储目录

## 手动编译步骤

### 前端编译

```bash
cd frontend
npm install
npm run build
```

构建产物位于 `embed/www/`

### 后端编译

在项目根目录执行：

```bash
go build -o imagespider .
```

## 开发模式

### 前端开发

```bash
cd frontend
npm run dev
```

访问 http://localhost:5173

API 会自动代理到 http://localhost:8080

### 后端开发

```bash
go run . -c config.yml
```

## Makefile（可选）

也可以创建 Makefile：

```makefile
.PHONY: all frontend backend clean run-dev docker-build

all: frontend backend

frontend:
	cd frontend && npm install && npm run build

backend:
	go build -o imagespider .

clean:
	rm -rf embed/www imagespider
	rm -rf frontend/node_modules

run-dev:
	cd frontend && npm run dev

docker-build:
	docker build -t imagespider:latest .

run:
	./imagespider -c config.yml
```

使用：
```bash
make          # 编译前端和后端
make frontend # 只编译前端
make backend  # 只编译后端
make clean    # 清理
make run      # 运行
```

## 常见问题

### 前端编译失败

1. 检查 Node.js 版本 >= 18
2. 删除 `frontend/node_modules` 重新安装：
   ```bash
   rm -rf frontend/node_modules
   cd frontend && npm install
   ```

### Go 编译失败

1. 检查 Go 版本 >= 1.21
2. 检查 CGO 相关依赖是否安装
3. 尝试关闭 CGO：
   ```bash
   CGO_ENABLED=0 go build -o imagespider .
   ```

### Docker 构建失败

1. 检查 Docker 版本 >= 20.10
2. 确保网络可以访问 npm 和 go 的镜像源
3. 使用国内镜像：
   ```bash
   # npm
   npm config set registry https://registry.npm.taobao.org

   # go
   go env -w GOPROXY=https://goproxy.cn,direct
   ```

## 目录结构说明

```
ImageSpider/
├── build.sh              # 编译脚本
├── Dockerfile            # Docker 构建文件
├── frontend/             # 前端源码
│   ├── src/              # 源代码
│   ├── dist/             # 构建产物（由 build.sh 生成）
│   └── node_modules/     # npm 依赖（由 build.sh 生成）
├── embed/
│   └── www/              # 前端构建产物（copy from frontend/dist）
├── imagespider           # 后端可执行文件（由 build.sh 生成）
└── config.yml            # 配置文件
```
