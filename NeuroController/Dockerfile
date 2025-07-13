# =========================
# 第一阶段：构建阶段（编译 Go 可执行文件）
# =========================
FROM golang:1.24.2-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制依赖文件并下载模块依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源码（包括 cmd/ internal/ web/ 等）
COPY . .

# 构建主控制器可执行文件
# GOARCH / GOOS 可通过 buildx 注入，实现多架构构建
RUN go build -o neurocontroller ./cmd/neurocontroller/main.go


# =========================
# 第二阶段：运行阶段（构建最终运行镜像）
# =========================
FROM alpine:latest

# 设置容器启动后的工作目录
WORKDIR /root/

# 从构建阶段复制已编译好的二进制文件
COPY --from=builder /app/neurocontroller .

# ✅ 复制前端静态页面目录（用于 /Atlhper 页面访问）
COPY --from=builder /app/web ./web

# ✅ 创建日志目录（防止未挂载卷时报错）
RUN mkdir -p /var/log/neurocontroller && chmod -R 755 /var/log/neurocontroller

# 赋予执行权限
RUN chmod +x ./neurocontroller

# ✅ 暴露 Gin 控制器的 Web 服务端口（用于 API + Webhook + UI）
EXPOSE 8081

# 容器启动命令
ENTRYPOINT ["./neurocontroller"]
