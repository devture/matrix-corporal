FROM golang:1.12.8-alpine3.10
WORKDIR /build
RUN apk --no-cache add git
COPY . /build/.
RUN go build matrix-corporal.go

FROM alpine:3.10
WORKDIR /
RUN apk --no-cache add ca-certificates
COPY --from=0 /build/matrix-corporal .
CMD ["./matrix-corporal"]
