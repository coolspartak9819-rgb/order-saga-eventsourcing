# Бойко Игорь Сергеевич

**Go Backend Developer / Backend Developer (Junior+ - Middle)**

Санкт-Петербург · предпочитаемый формат: удаленная работа

Telegram: `oneday2`

Email: `beatch1beatch@yandex.ru`

GitHub: https://github.com/coolspartak9819-rgb

## Цель

Ищу команду, в которой смогу развиваться как backend-разработчик на Go,
участвовать в разработке реального продукта, проходить код-ревью и постепенно
брать больше ответственности за сервисы и технические решения.

## О себе

Основной язык - Go. Разрабатываю backend-сервисы и веб-приложения, умею
разобраться в предметной области, спроектировать API, разделить код на слои и
довести проект до локального запуска через Docker.

Продолжаю системно изучать production-подходы: тестирование, конкурентный
доступ к данным, наблюдаемость, надежную работу интеграций и системный дизайн.
В работе ценю понятный код, обратную связь и возможность учиться у более
опытных коллег.

## Технические навыки

- Go: стандартная библиотека, `net/http`, `context`, goroutines, channels, `sync`
- Backend: REST API, gRPC, Protocol Buffers, JSON, middleware
- Данные: PostgreSQL, MySQL, Redis, `database/sql`, транзакции, индексы
- Архитектура: Event Sourcing, Saga, Outbox, идемпотентность, optimistic
  concurrency control
- Messaging: NATS JetStream, Apache Kafka, producer/consumer, event-driven подходы
- Тестирование: unit-тесты, table-driven tests, интеграционные сценарии
- Инфраструктура: Docker, Docker Compose, multi-stage builds, Linux, shell
- Практики: Git, GitHub Actions, логирование, метрики, диагностика сервисов
- Web: TypeScript, React, HTML, CSS
- System Design: декомпозиция сервисов, API-контракты, очереди, идемпотентность,
  consistency trade-offs

## Проекты

### Order Saga & Event Sourcing in Go

https://github.com/coolspartak9819-rgb/order-saga-eventsourcing

Учебный backend-проект для e-commerce сценария. Заказ восстанавливается из
цепочки доменных событий, а Saga-оркестратор координирует оплату и резервирование
товаров.

- Go HTTP API и доменный агрегат `Order`
- PostgreSQL и in-memory Event Store
- optimistic concurrency control и транзакционное сохранение событий
- Saga с компенсацией платежа при ошибке резервирования
- Outbox и публикация событий в NATS JetStream
- Kafka event bridge: чтение событий из NATS и запись в topic `orders.events`
- idempotency middleware, gRPC-клиенты, Docker Compose
- unit- и интеграционные тесты

### Другие проекты

- `food-delivery-saga` - сценарий доставки с Saga, gRPC/protobuf, PostgreSQL и
  Docker Compose: https://github.com/coolspartak9819-rgb/food-delivery-saga
- `booking-saga-platform` - сервисы бронирования, оплаты и билетов с общей
  оркестрацией: https://github.com/coolspartak9819-rgb/booking-saga-platform
- `notification-service` - gRPC-сервис уведомлений с Redis idempotency и
  обработкой событий: https://github.com/coolspartak9819-rgb/notification-service

## Образование

**Ставропольский государственный медицинский университет**

Специальность: лечебное дело

Год окончания: 2024

## Что ищу

Позиции: `Go Developer`, `Backend Developer`, `Junior+ / Middle Backend Engineer`.

Интересны продуктовые команды, где есть наставничество, код-ревью, инженерные
практики и возможность постепенно расти до самостоятельного middle-инженера.
