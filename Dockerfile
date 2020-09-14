FROM golang:1.15.2 AS build-env

RUN go get -u github.com/gobuffalo/packr/v2/packr2

WORKDIR /go/src/github.com/fairwindsops/goldilocks/
COPY . .
ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 packr2 build -a -o goldilocks *.go

FROM alpine:3.12.0 as alpine
RUN apk --no-cache --update add ca-certificates tzdata && update-ca-certificates

FROM scratch
COPY --from=alpine /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine /etc/passwd /etc/passwd

USER nobody
COPY --from=build-env /go/src/github.com/fairwindsops/goldilocks /

WORKDIR /opt/app

CMD ["/goldilocks"]
