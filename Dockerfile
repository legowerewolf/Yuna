FROM golang as builder
WORKDIR /go/src/yuna
COPY *.go ./
RUN go get ./...
RUN CGO_ENABLED=0 GOOS=linux go install -a

FROM scratch
COPY --from=builder /go/bin/yuna /yuna
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs
CMD ["yuna"]