FROM node:22-alpine AS admin_builder

WORKDIR /src/frontend/admin
COPY frontend/admin/package*.json ./
RUN npm ci
COPY frontend/admin/ ./
RUN npm run build

FROM golang:1.22-alpine AS builder

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=admin_builder /src/frontend/admin/dist ./internal/web/admin
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gravitylink ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache tzdata ca-certificates wget \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && adduser -D -H -u 10001 gravitylink \
    && mkdir -p /data \
    && chown gravitylink:gravitylink /data

# IP 地理库打进镜像（放在 /data 卷之外，避免被数据卷挂载遮蔽）。
# 构建上下文已有 xdb 时直接使用；否则从 ip2region 官方仓库下载 IPv4 库。
COPY backend/pkg/ipdb/ /tmp/ipdb-src/
RUN mkdir -p /app \
    && if [ -f /tmp/ipdb-src/ip2region.xdb ]; then \
         cp /tmp/ipdb-src/ip2region.xdb /app/ip2region.xdb; \
       elif [ -f /tmp/ipdb-src/ip2region_v4.xdb ]; then \
         cp /tmp/ipdb-src/ip2region_v4.xdb /app/ip2region.xdb; \
       else \
         wget -q -O /app/ip2region.xdb \
           https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb; \
       fi \
    && test -s /app/ip2region.xdb \
    && rm -rf /tmp/ipdb-src \
    && chown gravitylink:gravitylink /app/ip2region.xdb

ENV TZ=Asia/Shanghai \
    GEO_DB_PATH=/app/ip2region.xdb
USER gravitylink

WORKDIR /app
COPY --from=builder /out/gravitylink /app/gravitylink

EXPOSE 8080 8081
VOLUME ["/data"]
ENTRYPOINT ["/app/gravitylink"]
