# syntax=docker/dockerfile:1.7
# 多阶段构建：golang 编译 → alpine 精简运行镜像（约 10MB）
FROM golang:1.25-alpine AS build
# 通过 goproxy.cn 拉取依赖, 避免在无外网代理环境(如国内网络/受限内网)构建超时
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app .

FROM alpine:3.20
# 应用默认 shell 为 /bin/bash, alpine 基础镜像不带 bash, 缺失会导致
# 所有 /post 命令执行返回 "fork/exec /bin/bash: no such file" (HTTP 500)。
RUN apk add --no-cache bash \
    && addgroup -S app && adduser -S -G app app
COPY --from=build /out/app /usr/local/bin/app
USER app
EXPOSE 8080
ENTRYPOINT ["app"]
