FROM php:8.3-cli-alpine

WORKDIR /srv/app
COPY public /srv/app/public

RUN addgroup -S app && adduser -S app -G app \
    && chown -R app:app /srv/app

USER app
EXPOSE 8080
CMD ["php", "-S", "0.0.0.0:8080", "-t", "/srv/app/public"]
