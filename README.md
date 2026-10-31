# TASK1 — Инструменты автоматизации в Grafana

![alt text](<media/scheme.png>)

По этой схеме `dashboard-generator` автоматически создает дашборд в Grafana.

![alt text](media/dashboard.png)

На дашборде 9 графиков с golden signals для разных ручек. Целиком стенд для этой работы выглядит вот так:

![alt text](<media/overview.png>)

---

## Как запустить решение

Запустить стенд
```bash
cp .env.example .env
docker compose up -d --build
```

Посмотреть дашборд в браузере
```
http://localhost:3000/login
```

Остановить стенд
```bash
docker compose down -v
```

---

## Какие порты использовались

| Сервис | Порт | Назначение |
|---|---|---|
| app-msk | 8081 | Первый инстанс приложения |
| app-spb | 8082 | Второй инстанс приложения |
| prometheus | 9090 | Выполняет scrape обоих app |
| grafana | 3000 | Дашборд (логин и пароль `admin` либо в `.env`) |

---

## Какие метрики и лейблы были выбраны

| Метрика | Тип | Лейблы |
|---|---|---|
| `http_request_duration_seconds` | Histogram | `handler` + const `pod_id`, `datacenter` |
| `http_requests_total` | Counter | `handler`, `code` + const `pod_id`, `datacenter` |

**Почему histogram для latency:** перцентили (p50/p95/p99) считаются через `histogram_quantile` по сумме бакетов (`sum by (le)`). Так агрегация по нескольким подам/ДЦ остаётся корректной. Summary даёт квантили на инстансе, но их некорректно складывать между репликами.

---

## Приложение на go

Проверить работу приложения:
```bash
curl -sS -X POST http://localhost:8081/items \
  -H 'Content-Type: application/json' \
  -d '{"id":"1","name":"widget"}'

curl -sS http://localhost:8081/items/1

curl -sS -X PUT http://localhost:8081/items/1 \
  -H 'Content-Type: application/json' \
  -d '{"name":"widget-v2"}'
```

---

## Особенности проекта

Порядок запуска: `Apps` → `Prometheus` → `Grafana` → `Dashboard-generator` → `Load-generator`.

Конфиги Prometheus и Grafana лежат в `configs/` (bind-mount в compose).

Доли GET-запросов с ошибками `ERROR_404_RATE` и `ERROR_500_RATE` задаются в `.env`

Хранилище приложения сделано через in-memory CRUD. 

Без `POD_ID` и `DATACENTER` процесс не стартует.

Секреты через копирование `.env.example` в `.env`

---
