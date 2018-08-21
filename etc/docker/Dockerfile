FROM golang:1.10-alpine
WORKDIR /build
RUN apk --no-cache add git
RUN go get github.com/constabulary/gb/...
COPY . /build/.
RUN gb vendor restore
RUN gb build

FROM alpine:3.8
WORKDIR /
RUN apk --no-cache add ca-certificates
COPY --from=0 /build/bin/matrix-corporal .
CMD ["./matrix-corporal"]