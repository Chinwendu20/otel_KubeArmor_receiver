FROM alpine:3.16 as certs
RUN apk --update add ca-certificates

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chmod=755 otel-custom/otelcol-custom /otelcol-custom
COPY config.yml /etc/otelcol-custom/config.yaml
ENTRYPOINT ["/otel-custom"]
CMD ["--config", "/etc/otel-custom/config.yaml"]
EXPOSE 4317 55678 55679