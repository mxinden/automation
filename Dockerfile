FROM debian

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY ./automation /automation
CMD ["/automation"]
