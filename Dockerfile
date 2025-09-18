FROM golang:1.25.1 AS build

WORKDIR /
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 go build -o /go/bin/gomonitor

FROM scratch
COPY --from=build /go/bin/gomonitor /go/bin/gomonitor
COPY ./sample-big.json ./sample-big.json
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

CMD [ "/go/bin/gomonitor" ]
