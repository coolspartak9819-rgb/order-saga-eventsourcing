# Delivery Orders

Небольшое web-приложение для оформления и контроля заказов на доставку.
Проект сделан как production-style пример на ASP.NET 9 и React: заказ проходит
через API, сохраняется через Entity Framework Core в SQLite, а frontend показывает
рабочий dispatch board, а не только форму CRUD.

## Что умеет приложение

- создавать заказ с обязательными городами, адресами, весом и датой забора;
- автоматически выдавать номер формата `DLV-2026-A1B2C3`;
- отображать список заказов с поиском по номеру и городам;
- фильтровать заказы по статусу;
- открывать заказ в режиме чтения;
- показывать маршрут отправитель → получатель, детали груза и timeline;
- валидировать данные на backend и возвращать `ProblemDetails`;
- проверять состояние API и базы через `/health`;
- публиковать OpenAPI-документацию через `/openapi/v1.json`.

## Стек

- ASP.NET 9 Minimal API;
- Entity Framework Core 9;
- SQLite с volume для сохранения данных;
- React и Vite;
- nginx как frontend server и reverse proxy для `/api`;
- Docker Compose;
- GitHub Actions: backend build/test, frontend build и container build.

## Запуск через Docker

Нужны Docker и Docker Compose.

```bash
docker compose up --build
```

После запуска приложение доступно по адресу:

```text
http://localhost:8080
```

API и health check:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/orders
```

Остановить приложение:

```bash
docker compose down
```

SQLite хранится в Docker volume `delivery-orders-dotnet_orders-data`, поэтому
заказы не исчезают при обычном `docker compose down`.

## Локальный запуск без Docker

Нужны .NET SDK 9 и Node.js 22.

В первом терминале:

```bash
cd backend
dotnet run --urls http://localhost:5080
```

Во втором:

```bash
cd frontend
npm ci
npm run dev
```

Frontend будет доступен на `http://localhost:5173`.

## Проверки

```bash
dotnet restore
dotnet build -c Release
dotnet test tests/DeliveryOrders.Api.Tests/DeliveryOrders.Api.Tests.csproj -c Release

cd frontend
npm ci
npm run build
```

## API

| Метод | Endpoint | Назначение |
| --- | --- | --- |
| `GET` | `/api/orders` | список заказов; поддерживает `search` и `status` |
| `GET` | `/api/orders/{id}` | детальная информация о заказе |
| `POST` | `/api/orders` | создание заказа |
| `GET` | `/health` | health check API и SQLite |
| `GET` | `/openapi/v1.json` | OpenAPI-документ |

Пример создания:

```bash
curl -X POST http://localhost:8080/api/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "senderCity": "Saint Petersburg",
    "senderAddress": "Nevsky prospect, 28",
    "recipientCity": "Moscow",
    "recipientAddress": "Tverskaya street, 12",
    "weightKg": 2.5,
    "pickupDate": "2026-08-01"
  }'
```

## Архитектурные решения

Backend оставлен небольшим и прозрачным: Minimal API отвечает за HTTP-контракт,
EF Core за persistence, а генерация номера заказа вынесена в отдельный сервис.
У заказа есть UUID для внутренней связи и отдельный человекочитаемый номер для
оператора. Уникальный индекс на `OrderNumber` защищает от дублей на уровне базы.

Frontend использует боковую панель для создания и просмотра заказа, поэтому
оператор не теряет контекст списка. Маршрут и timeline дают быстрый визуальный
ответ на главный вопрос диспетчера: откуда, куда и на каком этапе находится
отправление.

Это демонстрационное приложение: статусы сейчас создаются в состоянии `Created`.
Следующим прикладным этапом для production была бы интеграция со службами
расчёта тарифа, трекинга и уведомлений.
