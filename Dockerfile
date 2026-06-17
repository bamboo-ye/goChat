FROM redis:7.4-alpine

WORKDIR /app

COPY .docker-build/gochat /app/gochat
COPY asset /app/asset
COPY config /app/config
COPY views /app/views
COPY index.html /app/index.html

RUN mkdir -p /app/asset/upload

EXPOSE 8081

ENTRYPOINT ["/app/gochat"]
CMD []
