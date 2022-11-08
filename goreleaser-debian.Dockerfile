FROM debian:latest
COPY xatu* /xatu
ENTRYPOINT ["/xatu"]
