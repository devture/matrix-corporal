FROM golang:1.14.2-alpine3.11
WORKDIR /build
RUN apk --no-cache add git
COPY . /build/.
RUN go get -u -v github.com/ahmetb/govvv
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "$(govvv -flags)" matrix-corporal.go

FROM alpine:3.11
WORKDIR /
RUN apk --no-cache add ca-certificates
COPY --from=0 /build/matrix-corporal .
CMD ["./matrix-corporal"]
