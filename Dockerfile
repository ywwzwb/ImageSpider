
# 第一阶段：编译前端
FROM node:20-alpine AS frontend-builder

# 安装 npm 依赖
RUN npm config set registry https://registry.npm.taobao.org

WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm install

# 复制前端源代码并编译
COPY frontend/ ./
RUN npm run build

# 第二阶段：编译 Go 后端
FROM golang:1.23-alpine AS backend-builder

# go env -w GOPROXY=https://goproxy.cn,direct,direct && \
# sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories && \
RUN go env -w CGO_ENABLED='0' && \
    apk update && apk add git

WORKDIR /go/src/imagespider/

# 复制 Go 源码和前端编译产物
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# 从前端编译阶段复制构建产物
COPY --from=frontend-builder /frontend/dist ./embed/www

RUN go build -o imagespider

# 第三阶段：运行阶段
FROM alpine:latest

# 第三阶段：运行阶段
FROM alpine:latest

# sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories &&\
RUN apk update && apk add \
    libheif-dev \
    x265-dev \
    jpeg-dev \
    libpng-dev \
    imagemagick \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# 创建应用目录
WORKDIR /app

# 从后端编译阶段复制二进制文件
COPY --from=backend-builder /go/src/imagespider/imagespider /app/imagespider

# 设置默认配置路径
ENV CONFIG_PATH=/config/config.yaml

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["/app/imagespider", "-c", "/config/config.yaml"]
