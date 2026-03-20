FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy go.work and all module files for dependency caching.
COPY go.work go.work.sum ./
COPY amnesia/go.mod amnesia/go.sum ./amnesia/
COPY api/go.mod api/go.sum ./api/
COPY cli/go.mod cli/go.sum ./cli/

RUN cd api && go mod download

# Copy source.
COPY amnesia/ ./amnesia/
COPY api/ ./api/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /zenv-api ./api/cmd/zenv-api

# ---

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /zenv-api /usr/local/bin/zenv-api
COPY api/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["zenv-api"]
