FROM golang:1.27-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS build

RUN apk add --no-cache ca-certificates git

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN export GOOS="${TARGETOS:-linux}"; \
    if [ -n "${TARGETARCH}" ]; then export GOARCH="${TARGETARCH}"; fi; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/webhook .

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/webhook /webhook

USER 65532:65532

ENTRYPOINT ["/webhook"]
