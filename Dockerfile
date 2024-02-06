FROM golang:1.21.1-bullseye AS builder
WORKDIR /build

ARG TARGETOS
ARG TARGETARCH

RUN apt update && apt install ca-certificates 

COPY . .

RUN go mod vendor

# Create statically linked server binary
RUN CGO_ENABLED=0 && GOOS=${TARGETOS} && GOARCH=${TARGETARCH} && go build -x -mod vendor -tags netgo -ldflags '-w -extldflags "-static"' -o ./bin/modem-logger ./cmd/main.go
# RUN CGO_ENABLED=0 && GOOS=${TARGETOS} && GOARCH=${TARGETARCH} && go build -x -tags netgo -ldflags '-w -extldflags "-static"' -o ./bin/modem-logger ./cmd/main.go


FROM scratch AS runner
WORKDIR /

COPY --from=builder /build/bin/modem-logger /bin/modem-logger
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

CMD ["modem-logger"]