FROM golang:1.17 as build

# Build
WORKDIR /root
COPY . .
RUN CGO_ENABLED=0 make

FROM alpine:3

RUN apk add --no-cache curl

HEALTHCHECK CMD curl --fail --silent http://localhost:26660/status || exit 1

# Copy binary
COPY --from=build /root/accumulated /bin/

ENTRYPOINT ["accumulated"]
CMD ["run", "--work-dir", "/node"]