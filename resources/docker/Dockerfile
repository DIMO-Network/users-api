FROM golang:1.20 AS build

RUN useradd -u 10001 dimo

WORKDIR /go/src/github.com/DIMO-Network/users-api/
COPY . /go/src/github.com/DIMO-Network/users-api/

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOFLAGS=-mod=vendor

RUN ls
RUN go mod tidy
RUN go mod vendor
RUN make install

FROM gcr.io/distroless/static-debian11 AS package

LABEL maintainer="DIMO <hello@dimo.zone>"

WORKDIR /

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /go/src/github.com/DIMO-Network/users-api/target/bin/users-api .
COPY --from=build /go/src/github.com/DIMO-Network/users-api/migrations /migrations

USER dimo

EXPOSE 8080

CMD ["/users-api"]
