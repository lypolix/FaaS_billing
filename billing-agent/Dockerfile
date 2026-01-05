# syntax=docker/dockerfile:1.7
FROM golang:1.25.1-alpine AS builder
WORKDIR /src

# лучше кэш для зависимостей
COPY go.mod go.sum ./
RUN go mod download

# исходники
COPY . .

# собираем весь пакет (все *.go в billing-agent/)
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/billing-agent .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates && update-ca-certificates
COPY --from=builder /out/billing-agent /usr/local/bin/billing-agent
ENTRYPOINT ["/usr/local/bin/billing-agent"]
