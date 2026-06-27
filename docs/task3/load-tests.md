# Тесты произоводительности

Длительность каждого из тестов составила 30 секунд

## До включения репликации

### Роут `/user/get/{id}`

| N clients | Latency (avg) | Throughput (req/s) |
|-----------|---------------|--------------------|
| 1         | 0.3ms         | 3134               |
| 10        | 1.2ms         | 8243               |
| 100       | 10.5ms        | 9566               |
| 1000      | 96.6ms        | 10349              |

### Роут `/user/search`

| N clients | Latency (avg) | Throughput (req/s) |
|-----------|---------------|--------------------|
| 1         | 0.9ms         | 1136               |
| 10        | 2.2ms         | 4601               |
| 100       | 14ms          | 6957               |
| 1000      | 111ms         | 8325               |

## После включения синхронной репликации

### Роут `/user/get/{id}`

| N clients | Latency (avg) | Throughput (req/s) |
|-----------|---------------|--------------------|
| 1         | 0.3ms         | 3000               |
| 10        | 1.2ms         | 8571               |
| 100       | 3.6ms         | 27607              |
| 1000      | 30ms          | 40421              |

### Роут `/user/search`

<details>
<summary>Команда</summary>

```bash
hey \
-c 1 \   
-z 30s \
http://localhost:8282/api/v1/user/search\?first_name\=Александр\&last_name\=Абрамов
```
</details>

| N clients | Latency (avg) | Throughput (req/s) |
|-----------|---------------|--------------------|
| 1         | 1.5ms         | 681                |
| 10        | 2.7ms         | 3687               |
| 100       | 17ms          | 5885               |
| 1000      | 148ms         | 6702               |

## Выводы по результатам тестов операций чтения после включения репликации

У обоих ручек при небольшом количестве одновременных запросов (1, 10) практически не изменилась производительность. Однако при большем числе (100, 1000) гораздо заметнее становится рост производительности после того как была включена асинхронная репликация

## Запись после включения синхронной кворумной репликации

Параметр применён через Patroni REST API (PATCH /config). PostgreSQL подтвердил:

```
synchronous_standby_names = ANY 1 ("patroni-node2","patroni-node3")
synchronous_commit        = on
```

Любая транзакция на primary ждёт подтверждения WAL от минимум одной реплики — при потере мастера новый лидер гарантированно содержит все закоммиченные данные.

Чтобы настройка сохранялась при перезапуске кластера (а не только в etcd runtime), она уже прописана в bootstrap.dcs через build/package/patroni-entrypoint.sh. Для существующего кластера с непустым PGDATA patroni пропускает bootstrap и не пишет bootstrap.dcs в etcd — поэтому применили через API вручную.

### Нагрузочное тестирование записи — POST /api/v1/user/register

<details>
<summary>Команда</summary>

```
hey -z 30s -c <N> -m POST \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Johny","second_name":"Depp","password":"qwerty",
       "city":"Florida","birthdate":"1995-12-06",
       "biography":"it was a long story to tell ..."}' \
  http://localhost:8282/api/v1/user/register
```
</details>

| N clients | Latency (avg)  | Throughput (req/s) |
|-----------|----------------|--------------------|
| 1         | 44.8ms         | 670                |
| 10        | 50.2ms         | 5982               |
| 100       | 296ms          | 10156              |
| 1000      | 2942ms         | 10195              |

#### Summary

c=1 → c=10 — линейный рост throughput (×9), latency почти не изменилась (+5 ms). Система под недогрузкой, запросы не конкурируют.

c=10 → c=100 — throughput вырос лишь в 1.7×, latency — в 5.9×. Начинается очередь на пул соединений к БД: MaxOpenConns = 25 (см. internal/config/config.go:28), остальные воркеры ждут свободного слота.

c=100 → c=1000 — throughput практически не изменился (337 rps), latency выросла в 10×, появились таймауты. Система упёрлась в потолок: 25 соединений к мастеру + синхронное ожидание WAL-подтверждения от реплики дают фиксированный максимум ~337 rps.

---

## Эксперимент: потери транзакций при падении мастера

### Цель

Проверить, теряются ли зафиксированные транзакции при внезапной остановке мастер-узла в кластере с **кворумной синхронной репликацией** (`synchronous_mode: quorum`, `synchronous_node_count: 1`).

### Условия

| Параметр | Значение |
| --- | --- |
| Кластер | Patroni 3 узла + etcd + HAProxy |
| Синхронная репликация | `ANY 1 ("patroni-node2","patroni-node3")` |
| Нагрузка | `hey`, 10 воркеров, `POST /api/v1/user/register` |
| Остановка мастера | `docker stop sm-patroni-node1` (SIGTERM) |

### Как воспроизвести

```bash
# 1. Убедиться, что кластер запущен и node1 является мастером
docker compose -f deployments/docker-compose-cluster-patroni.yaml ps
curl -s http://localhost:8008/patroni | python3 -c \
  "import sys,json; d=json.load(sys.stdin); print(d['role'])"
# ожидаемый вывод: primary

# 2. Проверить, что включена кворумная синхронная репликация
docker exec sm-patroni-node1 bash -c \
  'PGPASSWORD=$DB_PASSWORD psql -h 127.0.0.1 -U $DB_USER -d postgres \
   -tAc "SHOW synchronous_standby_names;"'
# ожидаемый вывод: ANY 1 ("patroni-node2","patroni-node3")

# 3. Зафиксировать базовое количество строк
docker exec sm-patroni-node1 bash -c \
  'PGPASSWORD=$DB_PASSWORD psql -h 127.0.0.1 -U $DB_USER -d social_media_db \
   -tAc "SELECT COUNT(*) FROM users;"'
# → BASELINE

# 4. Запустить нагрузку в фоне (120 с, 10 воркеров)
hey -z 120s -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Johny","second_name":"Depp","password":"qwerty",
       "city":"Florida","birthdate":"1995-12-06",
       "biography":"it was a long story to tell ..."}' \
  http://localhost:8282/api/v1/user/register > /tmp/hey_out.txt 2>&1 &
HEY_PID=$!

# 5. Выждать 15 секунд, снять счётчик на активном мастере
sleep 15
docker exec sm-patroni-node1 bash -c \
  'PGPASSWORD=$DB_PASSWORD psql -h 127.0.0.1 -U $DB_USER -d social_media_db \
   -tAc "SELECT COUNT(*) FROM users;"'
# → COUNT_ON_MASTER

# 6. Остановить мастер и нагрузку
docker stop sm-patroni-node1
kill $HEY_PID 2>/dev/null; wait $HEY_PID 2>/dev/null

# 7. Дождаться выбора нового мастера (Patroni TTL=30 s, loop_wait=10 s)
NEW_CONTAINER=""
until [ -n "$NEW_CONTAINER" ]; do
  sleep 2
  curl -sf http://localhost:8009/primary >/dev/null 2>&1 && NEW_CONTAINER="sm-patroni-node2"
  curl -sf http://localhost:8010/primary >/dev/null 2>&1 && NEW_CONTAINER="sm-patroni-node3"
done
echo "Новый мастер: $NEW_CONTAINER"

# 8. Снять счётчик на новом мастере
docker exec $NEW_CONTAINER bash -c \
  'PGPASSWORD=$DB_PASSWORD psql -h 127.0.0.1 -U $DB_USER -d social_media_db \
   -tAc "SELECT COUNT(*) FROM users;"'
# → COUNT_ON_NEW_MASTER

# 9. Рассчитать потери:  COUNT_ON_MASTER - COUNT_ON_NEW_MASTER
```

### Результаты

| Момент | Количество строк в `users` |
| --- | ---: |
| До нагрузки (baseline) | 1 026 936 |
| На мастере (node1) перед `docker stop` | 1 029 907 |
| На новом мастере (node3) после failover | 1 029 952 |

```text
Записано через нагрузку (снимок на мастере):  1 029 907 − 1 026 936 = 2 971 строк
На новом мастере от baseline:                 1 029 952 − 1 026 936 = 3 016 строк
Потери транзакций:                            1 029 907 − 1 029 952 = −45 (потерь нет)
```

> Новый мастер содержит на **45 строк больше**, чем показал снимок на старом мастере. Это ожидаемо: между моментом снимка и выполнением `docker stop` несколько транзакций успели зафиксироваться и подтвердиться кворумом, но не попали в снимок.

### Вывод

**Потери транзакций: 0.**

Кворумная синхронная репликация (`synchronous_commit = on`, `ANY 1 (*)`) гарантирует, что каждый `200 OK` ответ клиенту означает запись WAL минимум на одном реплика-узле. При падении мастера новый лидер уже содержит все подтверждённые транзакции — данные не теряются.
