FROM golang:1.12 AS build-env
WORKDIR /go/src/github.com/fairwindsops/vpa-analysis/

COPY . .
RUN make build

FROM alpine:3.9
WORKDIR /usr/local/bin
RUN apk --no-cache add ca-certificates

USER nobody
COPY --from=build-env /go/src/github.com/fairwindsops/vpa-analysis/vpa-analysis .

ENTRYPOINT ["vpa-analysis"]
CMD ["controller", "-v", "2"]
