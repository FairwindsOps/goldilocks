FROM golang:1.12.4 AS build-env
WORKDIR /go/src/github.com/fairwindsops/goldilocks/

ENV GO111MODULE=on
COPY . .
RUN go get -u github.com/gobuffalo/packr/v2/packr2
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 packr2 build -a -o goldilocks *.go

FROM alpine:3.9
WORKDIR /usr/local/bin
RUN apk --no-cache add ca-certificates

RUN addgroup -S goldilocks && adduser -u 1200 -S goldilocks -G goldilocks
USER 1200
COPY --from=build-env /go/src/github.com/fairwindsops/goldilocks /

WORKDIR /opt/app

CMD ["/goldilocks"]
