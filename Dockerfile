FROM golang:1.21.6-alpine AS base
WORKDIR /app

# builder
FROM base AS builder
ENV GOOS linux
ENV GOARCH amd64

# build-args
ARG VERSION

RUN apk --no-cache add bash git openssh

# modules: utilize build cache
COPY go.mod ./
COPY go.sum ./

# RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY . .

# add gcc for cgo
RUN apk add --no-cache gcc musl-dev

# inject versioning information & build the binary
# appsec: datadog ASM
RUN export BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ"); export CGO_ENABLED=1; go build -v -tags appsec -o backend -ldflags "-X exusiai.dev/backend-next/internal/pkg/bininfo.Version=$VERSION -X exusiai.dev/backend-next/internal/pkg/bininfo.BuildTime=$BUILD_TIME" .

# runner
FROM base AS runner
RUN apk add --no-cache libc6-compat tini
# Tini is now available at /sbin/tini

COPY --from=builder /app/backend /app/backend
EXPOSE 8080

ENTRYPOINT ["/sbin/tini", "--"]
CMD [ "/app/backend", "start" ]
