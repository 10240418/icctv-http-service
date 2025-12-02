# 构建阶段
FROM docker.1ms.run/golang:1.24 AS builder

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 禁用代理并下载依赖（在同一个 RUN 中执行，确保环境变量生效）
RUN unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy && \
  export HTTP_PROXY="" && \
  export HTTPS_PROXY="" && \
  export http_proxy="" && \
  export https_proxy="" && \
  export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct && \
  go mod download

# 复制源代码
COPY . .

# 编译（CGO_ENABLED=0 生成静态二进制文件，适合多架构）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/main ./main.go

# 运行阶段 - 使用 scratch（最小镜像，适合静态编译的 Go 程序）
FROM scratch

# 从构建阶段复制二进制文件
COPY --from=builder /app/main /main

# 暴露端口
EXPOSE 8080

# 运行服务
CMD ["/main"]

