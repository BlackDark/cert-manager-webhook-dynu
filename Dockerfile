FROM golang:1.27-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS build

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
