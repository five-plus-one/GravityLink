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

RUN adduser -D -H -u 10001 gravitylink
USER gravitylink

WORKDIR /app
COPY --from=builder /out/gravitylink /app/gravitylink

EXPOSE 8080 8081
ENTRYPOINT ["/app/gravitylink"]
