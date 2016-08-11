FROM golang:1.6

COPY . /go/src/github.com/RealImage/udt-tunnel

WORKDIR /go/src/github.com/RealImage/udt-tunnel/vendor/github.com/xjdrew/go-udtwrapper

RUN make -e arch=AMD64 && cp ./udt/libudt.a /go/src/github.com/RealImage/udt-tunnel

WORKDIR /go/src/github.com/RealImage/udt-tunnel

RUN go install

ENTRYPOINT udt-tunnel
