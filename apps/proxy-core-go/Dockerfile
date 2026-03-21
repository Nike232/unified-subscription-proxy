FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o proxy-core-go ./apps/proxy-core-go

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/proxy-core-go .
EXPOSE 8081
CMD ["./proxy-core-go"]
