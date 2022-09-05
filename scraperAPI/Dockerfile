# syntax=docker/dockerfile:1
#Alpine based image for smaller footprint
FROM golang:1.15.6-alpine

#Signature
LABEL author="Jerin Thomas www.jerin-thomas.com"

ENV APP_HOME $GOPATH/src/scraperAPI
WORKDIR "$APP_HOME"

COPY . .
RUN go mod download
RUN go build -o scraperAPI
EXPOSE 8080
CMD ["./scraperAPI"]
