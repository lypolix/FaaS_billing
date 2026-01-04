# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src

# git нужен, если есть зависимости через VCS
RUN apk add --no-cache git ca-certificates

# Сначала зависимости (быстрее кешируется)
COPY go.mod go.sum ./
RUN go mod download

# Потом исходники
COPY . .

# Собираем backend cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/backend


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=build /out/backend /app/backend

# Gin у тебя слушает addr из env BACKENDADDR, по умолчанию 8080 [file:30]
EXPOSE 8080

ENV BACKENDADDR=":8080"

ENTRYPOINT ["/app/backend"]
