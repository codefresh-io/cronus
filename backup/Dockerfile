FROM alpine:3.11

RUN apk add --no-cache curl ca-certificates openssl

COPY backup.sh /backup.sh
RUN chmod +x /backup.sh

CMD ["/backup.sh"]