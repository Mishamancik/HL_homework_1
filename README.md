# Items API — мониторинг золотых сигналов

Go CRUD с Prometheus-метриками, двумя инстансами за compose, автогенерацией Grafana-дашборда и фоновой нагрузкой.

## Быстрый старт

```bash
cp .env.example .env
docker compose up -d --build
```

Через 2–3 минуты: Grafana → дашборд **Items API — golden signals**, данные под нагрузкой.

Остановка: `docker compose down`

## Порты

| Сервис | Порт | Назначение |
|---|---|---|
| app-msk (`pod-msk-1`) | 8081 | Первый инстанс приложения (CRUD + `/metrics`) |
| app-spb (`pod-spb-1`) | 8082 | Второй инстанс приложения |
| Prometheus | 9090 | scrape обоих app |
| Grafana | 3000 | дашборд (логин и пароль `admin` либо задать свой через `.env`) |

Секреты только в `.env` (см. `.env.example`, `.env` в `.gitignore`).

## Ручки приложения

| Метод | Путь | Метрики |
|---|---|---|
| `POST` | `/items` | да |
| `GET` | `/items/{id}` | да |
| `PUT` | `/items/{id}` | да |
| `GET` | `/health` | нет |
| `GET` | `/metrics` | нет (Prometheus scrape) |
| `GET` | `/handlers` | нет (список бизнес-ручек для генератора) |

Хранилище — in-memory. Без `POD_ID` и `DATACENTER` процесс не стартует.

## Один образ — три бинарника

`Dockerfile` собирает образ `hometask1-app:local` с тремя бинарями:

| Бинарь | Роль | Как выбирается в compose |
|---|---|---|
| `app` | HTTP-сервис | entrypoint по умолчанию (`app-msk`, `app-spb`) |
| `dashboard-gen` | тянет `/handlers`, публикует non-editable дашборд | `entrypoint: dashboard-gen` |
| `loadgen` | ramp до ~20–50 RPS на оба инстанса | `entrypoint: loadgen` |

Один build — разные сервисы просто подменяют, что запускать.

Порядок: apps → Prometheus → Grafana → `dashboard-gen` (once) → `load` (пока стек жив).

Нагрузка: ~3% GET на missing id → 404; `FAULT_500_RATE` в app → ~3% 500.

## Метрики

| Метрика | Тип | Лейблы |
|---|---|---|
| `http_request_duration_seconds` | **Histogram** | `handler` + const `pod_id`, `datacenter` |
| `http_requests_total` | Counter | `handler`, `code` + const `pod_id`, `datacenter` |

`handler` — шаблон вида `GET /items/{id}` (не фактический URL).

**Почему histogram для latency:** перцентили (p50/p95/p99) считаются через `histogram_quantile` по сумме бакетов (`sum by (le)`). Так агрегация по нескольким подам/ДЦ остаётся корректной. Summary даёт квантили на инстансе, но их некорректно складывать между репликами.

RPS: `rate(http_requests_total[…])`.  
Ошибки на дашборде: доля 4xx/5xx по `code` от всех запросов ручки.
