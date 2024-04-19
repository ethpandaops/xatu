FROM gcr.io/distroless/cc-debian12:latest
COPY xatu* /xatu
ENTRYPOINT ["/xatu"]
