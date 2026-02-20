FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_CFLAGS="-Wno-discarded-qualifiers" go build -o /out/tiny-headend .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/tiny-headend /usr/local/bin/tiny-headend

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/tiny-headend"]
