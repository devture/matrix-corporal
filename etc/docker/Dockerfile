FROM docker.io/golang:1.16.5-alpine3.14
WORKDIR /build
RUN apk --no-cache add git
COPY . /build/.
RUN go get -u -v github.com/ahmetb/govvv
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "$(govvv -flags)" matrix-corporal.go

FROM docker.io/alpine:3.14
WORKDIR /
RUN apk --no-cache add ca-certificates
COPY --from=0 /build/matrix-corporal .
CMD ["./matrix-corporal"]
HEALTHCHECK CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:41080/_matrix/client/corporal || exit 1
