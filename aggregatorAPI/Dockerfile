# syntax=docker/dockerfile:1
#Alpine based image for smaller footprint
FROM golang:1.15.6-alpine 

#Signature
LABEL author="Jerin Thomas www.jerin-thomas.com"

ENV APP_HOME $GOPATH/aggregatorAPI
WORKDIR "$APP_HOME"

COPY . .
RUN go mod download
RUN go build -o /aggregatorAPI
EXPOSE 8081
CMD ["./aggregatorAPI"]