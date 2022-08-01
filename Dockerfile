FROM golang:1.18-buster AS build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.com.cn,direct
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /exporter

FROM alpine:3.16
WORKDIR /
COPY --from=build /exporter /exporter
EXPOSE 80
ENTRYPOINT ["/exporter"]