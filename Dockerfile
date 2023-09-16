
FROM golang:1.21-alpine AS build-env

ENV PACKAGES curl make git libc-dev bash gcc linux-headers eudev-dev
RUN apk add --no-cache $PACKAGES

WORKDIR /abci-workshop

COPY go.mod go.sum ./

RUN go mod download
COPY . .

RUN make install
#

FROM alpine:3

EXPOSE 26656 26657 1317 9090 6060

WORKDIR /root

# Install minimum necessary dependencies
RUN apk add --no-cache curl make bash jq sed

# Copy over binaries from the build-env
COPY --from=build-env /abci-workshop/build/cosmappd /usr/bin/cosmappd

ENTRYPOINT ["/bin/bash"]

CMD ["cosmappd"]
