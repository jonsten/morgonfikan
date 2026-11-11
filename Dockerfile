FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/morgonfikan ./cmd/morgonfikan

FROM scratch

COPY --from=build /out/morgonfikan /morgonfikan
LABEL org.opencontainers.image.source="https://github.com/jonsten/morgonfikan"
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/morgonfikan"]
