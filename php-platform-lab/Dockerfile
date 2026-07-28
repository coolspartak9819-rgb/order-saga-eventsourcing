FROM php:8.3-fpm-alpine

RUN apk add --no-cache librdkafka \
    && apk add --no-cache --virtual .build-deps $PHPIZE_DEPS librdkafka-dev \
    && docker-php-ext-install pdo_mysql \
    && pecl install rdkafka \
    && docker-php-ext-enable rdkafka \
    && apk del .build-deps

WORKDIR /var/www/html
COPY app/ /var/www/html/

EXPOSE 9000
CMD ["php-fpm", "-F"]
