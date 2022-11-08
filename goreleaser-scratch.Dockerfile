FROM gcr.io/distroless/static-debian11:latest
COPY xatu* /xatu
ENTRYPOINT ["/xatu"]
