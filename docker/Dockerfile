FROM golang:alpine AS builder
ENV CGO_ENABLED=1
WORKDIR /var/tmp
RUN set -ex \
    && apk update \
    && apk add git build-base \
    && git clone https://cgit.bbaa.fun/bbaa/go-pastebin.git \
    && cd go-pastebin \
    && go build -v -o go-pastebin -trimpath -ldflags "-s -w -buildid=" ./cmd
FROM alpine:latest
COPY --from=builder /var/tmp/go-pastebin/go-pastebin /usr/bin/go-pastebin
RUN set -ex \
  && apk update \
  && apk add --no-cache tzdata ca-certificates mailcap \
  && mkdir -p /var/lib/go-pastebin \
  && chmod +x /usr/bin/go-pastebin
WORKDIR /var/lib/go-pastebin
VOLUME /var/lib/go-pastebin
ENV TZ=Asia/Shanghai
EXPOSE 8080
CMD [ "/usr/bin/go-pastebin" ]
