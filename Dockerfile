FROM golang:1.14

WORKDIR /go/src/turnin-compute
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["turnin-compute"]