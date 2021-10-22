FROM alpine:3.14

# 'nobody' user in alpine
USER 65534
COPY goldilocks /

CMD ["/goldilocks"]
