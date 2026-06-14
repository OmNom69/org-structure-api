FROM golang:1.26.1-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server ./cmd/app

FROM alpine:3.22
WORKDIR /app

COPY --from=builder /app/server .
EXPOSE 8080

CMD ["./server"]