# 构建阶段
FROM node:18-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制package.json和package-lock.json
COPY package*.json ./

# 安装依赖
RUN npm install --silent

# 复制源代码
COPY . .

# 构建应用
RUN npm run build

# 运行阶段 - 使用nginx提供静态内容
FROM nginx:alpine

# 复制构建产物到nginx服务目录
COPY --from=builder /app/build /usr/share/nginx/html

# 复制nginx配置文件
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 暴露端口
EXPOSE 80

# 启动nginx
CMD ["nginx", "-g", "daemon off;"] 