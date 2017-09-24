FROM scratch
EXPOSE 3000

COPY build/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY build/httppub /app/httppub
ENTRYPOINT ["/app/httppub"]
