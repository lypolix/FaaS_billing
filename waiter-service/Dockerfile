FROM golang:1.25.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o waiter main.go

FROM alpine:3.20
RUN apk --no-cache add --no-check-certificate ca-certificates && update-ca-certificates
WORKDIR /root/
COPY --from=builder /app/waiter .
EXPOSE 8080
CMD ["./waiter"]
