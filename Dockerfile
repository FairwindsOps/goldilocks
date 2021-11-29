FROM alpine:3.15.0

# 'nobody' user in alpine
USER 65534
COPY goldilocks /

CMD ["/goldilocks"]
