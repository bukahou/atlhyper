# =========================
# 第一阶段：构建阶段
# =========================
FROM golang:1.21-alpine AS builder

# 安装 Git（go mod 可能会拉取）
# RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制模块定义并拉取依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码
COPY . .

# 构建目标控制器程序
RUN go build -o neurocontroller ./cmd/neurocontroller/main.go

# =========================
# 第二阶段：运行阶段
# =========================
FROM alpine:latest

# 设置工作目录
WORKDIR /root/

# 拷贝编译后的二进制文件
COPY --from=builder /app/neurocontroller .

# 可选：复制配置文件（例如 config.yaml）
# COPY internal/config/config.yaml ./config.yaml
# COPY internal/config/webhook_map.yaml ./webhook_map.yaml
# COPY internal/config/mail_template.tmpl ./mail_template.tmpl

# 暴露端口（Gin 默认 8080）
# EXPOSE 8080

# 启动命令
ENTRYPOINT ["./neurocontroller"]
