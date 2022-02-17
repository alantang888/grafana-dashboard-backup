FROM golang:1.17 as build

WORKDIR /go/src/github.com/alantang888/grafana-dashboard-backup
COPY . .
ENV GO111MODULE=on
RUN go mod download
WORKDIR /go/src/github.com/alantang888/grafana-dashboard-backup/cmd/grafana-dashboard-backup
RUN go build -o /go/bin/app


FROM gcr.io/distroless/base
COPY --from=build /go/bin/app /

CMD ["/app"]
