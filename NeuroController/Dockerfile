# =========================
# Stage 1: Build Phase
# =========================
FROM golang:1.24.2-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the target controller binary
# GOARCH and GOOS will be injected by buildx during multi-platform builds
RUN go build -o neurocontroller ./cmd/neurocontroller/main.go


# =========================
# Stage 2: Runtime Phase
# =========================
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the compiled binary from the build stage
COPY --from=builder /app/neurocontroller .

# ✅ Create log directory for deployment (prevent crash if volume is not mounted)
RUN mkdir -p /var/log/neurocontroller && chmod -R 755 /var/log/neurocontroller

# Make the binary executable
RUN chmod +x ./neurocontroller

# ✅ 明确暴露 Gin Webhook 服务端口
EXPOSE 8081

# Default startup command
ENTRYPOINT ["./neurocontroller"]

