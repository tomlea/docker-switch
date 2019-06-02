FROM golang:1.12

WORKDIR /go/src/app
COPY . .

RUN go build
RUN ln -s /go/src/app/app /bin/dockerswitch

VOLUME /data
