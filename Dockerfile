FROM alpine:3.15.3

LABEL org.opencontainers.image.authors="FairwindsOps, Inc." \
      org.opencontainers.image.vendor="FairwindsOps, Inc." \
      org.opencontainers.image.title="goldilocks" \
      org.opencontainers.image.description="Goldilocks is a utility that can help you identify a starting point for resource requests and limits." \
      org.opencontainers.image.documentation="https://goldilocks.docs.fairwinds.com/" \
      org.opencontainers.image.source="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.url="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.licenses="Apache License 2.0"

# 'nobody' user in alpine
USER 65534
COPY goldilocks /

CMD ["/goldilocks"]
