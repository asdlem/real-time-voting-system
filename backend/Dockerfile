# 构建阶段
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /build

# 设置环境变量以使用国内Go模块代理
ENV GOPROXY=https://goproxy.cn,direct

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# 运行阶段
FROM alpine:latest

# 安装基本工具
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建目录
WORKDIR /app

# 从构建阶段复制编译好的应用
COPY --from=builder /build/server .

# 暴露端口
EXPOSE 8090

# 设置运行时环境变量
ENV GIN_MODE=release
ENV SERVER_PORT=8090
ENV API_PREFIX=/api

# 启动应用
CMD ["./server"]
