FROM golang:1.19 as builder
RUN echo ${TARGETARCH}
RUN mkdir -p /build
COPY ./ /build/
WORKDIR /build/

FROM builder AS builder-amd64
RUN env GOOS=linux GOARCH=amd64 make clean bin/theunwrapper

FROM builder AS builder-arm64
RUN env GOOS=linux GOARCH=arm64 make clean bin/theunwrapper

FROM builder-${TARGETARCH} AS build

FROM alpine:latest
RUN apk add gcompat
COPY --from=build /build/bin/theunwrapper /bin/theunwrapper
ENTRYPOINT ["/bin/theunwrapper", "-log-format=pretty"]
