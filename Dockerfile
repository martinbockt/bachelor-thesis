FROM golang:1.22.5-alpine AS BuildStage

WORKDIR /app
COPY . .

RUN go mod vendor
RUN go build -o /llmscraper cmd/api/main.go

FROM alpine:latest

RUN apk add chromium

WORKDIR /
COPY --from=BuildStage /llmscraper /llmscraper

EXPOSE 8080
ENTRYPOINT [ "/llmscraper" ]