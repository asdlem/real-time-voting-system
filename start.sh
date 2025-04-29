#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}==== 实时投票系统启动脚本 ====${NC}"

# 检查docker和docker-compose是否安装
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Docker未安装，请先安装Docker${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}Docker Compose未安装，请先安装Docker Compose${NC}"
    exit 1
fi

# 启动服务
echo -e "${GREEN}启动Docker容器...${NC}"
docker-compose up -d

# 等待服务启动
echo -e "${GREEN}等待服务启动...${NC}"
sleep 5

# 检查服务状态
echo -e "${GREEN}检查服务状态:${NC}"
docker-compose ps

echo -e "${GREEN}==== 服务启动完成 ====${NC}"
echo -e "${GREEN}前端访问地址: http://localhost${NC}"
echo -e "${GREEN}后端API地址: http://localhost:8090/api${NC}" 