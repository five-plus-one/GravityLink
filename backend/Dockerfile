FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gravitylink ./cmd/server

FROM alpine:3.20

RUN adduser -D -H -u 10001 gravitylink
USER gravitylink

WORKDIR /app
COPY --from=builder /out/gravitylink /app/gravitylink

EXPOSE 8080
ENTRYPOINT ["/app/gravitylink"]
