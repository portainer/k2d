FROM alpine:latest as base
RUN apk add -U --no-cache ca-certificates

FROM scratch
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY dist /

ENTRYPOINT [ "/k2d" ]