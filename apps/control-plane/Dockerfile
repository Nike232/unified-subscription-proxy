FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o control-plane ./apps/control-plane

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/control-plane .
COPY --from=builder /app/data ./data
EXPOSE 8080
CMD ["./control-plane"]
