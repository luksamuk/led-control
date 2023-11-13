FROM golang:1.20 as builder
RUN mkdir /project
WORKDIR /project
ENV CGO_ENABLED=0 \
    GOOS=linux
COPY . .
RUN go build -o ledsvc.linux.amd64 cmd/ledsvc/ledsvc.go

FROM alpine:latest as certs
RUN apk --no-cache add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /project/ledsvc.linux.amd64 ledsvc
EXPOSE 2112
ENTRYPOINT ["./ledsvc"]
