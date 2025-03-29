# syntax=docker/dockerfile:1

FROM 1.23.7-alpine3.20
WORKDIR /go/src/github.com/pennsieve/pennsieve-agent/
COPY go.mod .
COPY go.sum .
RUN apk add build-base
RUN go mod download
RUN go install github.com/mattn/go-sqlite3
COPY . .
RUN CGO_ENABLED=1 go build -a -installsuffix cgo -o pennsieve .

FROM alpine:3.20
WORKDIR /root/
RUN apk update && apk upgrade
RUN apk add --no-cache sqlite
COPY --from=0 /go/src/github.com/pennsieve/pennsieve-agent/pennsieve ./
EXPOSE 9000
CMD ["./pennsieve", "agent", "startgit "]