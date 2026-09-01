# syntax=docker/dockerfile:1.7
# 多阶段构建：golang 编译 → alpine 精简运行镜像（约 10MB）
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app .

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/app /usr/local/bin/app
USER app
EXPOSE 8080
ENTRYPOINT ["app"]
