# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/backend


FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/backend /backend

EXPOSE 8080
ENV BACKENDADDR=":8080"
ENTRYPOINT ["/backend"]
