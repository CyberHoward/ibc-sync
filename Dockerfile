#--- Build stage
FROM golang:1.21-alpine3.17 AS go-builder

WORKDIR /src

# hadolint ignore=DL4006
RUN set -eux \
    && apk add --no-cache ca-certificates build-base git linux-headers
COPY . /src/

RUN BUILD_TAGS=muslc LINK_STATICALLY=true make build

#--- Image stage
FROM alpine:3.17.3

COPY --from=go-builder /src/build/cosmappd /usr/bin/cosmappd

# Set up dependencies
ENV PACKAGES curl make bash jq sed

# Install minimum necessary dependencies
RUN apk add --no-cache $PACKAGES

WORKDIR /opt

ENTRYPOINT ["cosmappd"]