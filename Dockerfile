FROM golang:1.26.6

WORKDIR /usr/src/app

COPY ./src/go.mod ./
RUN go mod tidy
RUN go mod download

COPY ./src .
EXPOSE 8787
RUN go build -v -o /usr/local/bin/app ./...

CMD ["app"]
