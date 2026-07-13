# Шардирование

Пошаговое руководство по горизонтальному шардированию таблицы сообщений
(`dialog_message`) с помощью **Citus** и с возможностью **решардинга без
даунтайма**.

## Зачем шардировать

`dialog_message` — самая нагруженная на запись и чтение таблица (одна строка на
каждое сообщение чата). Реплики (задача 4) масштабируют только чтение — запись
по-прежнему упирается в один мастер. Шардирование распределяет и данные, и
нагрузку на запись по нескольким узлам.

Citus — расширение PostgreSQL, превращающее его в распределённую БД:
- **coordinator** — единая точка входа, куда ходит приложение; хранит метаданные
  и маршрутизирует/распараллеливает запросы;
- **worker** — узлы, на которых лежат шарды (shard = обычная таблица Postgres);
- приложение работает **только с координатором** и не знает о шардах, поэтому
  Go-код запросов (`internal/repository/dialog.go`) менять не нужно.

## Выбор ключа шардирования

Ключ шардирования (distribution column) определяет, на какой шард попадёт строка
(`shard = hash(ключ) mod shard_count`). Для доменной модели диалогов есть три
варианта. Ключевые запросы приложения:
- `ListMessages` → `WHERE dialog_id = $1` (чтение переписки);
- `GetDialogID` → поиск диалога по паре пользователей через `dialog_user`.

| Вариант | Как распределяем | `ListMessages` | `GetDialogID` | Вывод |
|---|---|---|---|---|
| **`dialog_id` + reference-таблицы** ✅ | `dialog_message` по `dialog_id`; `users` / `dialog` / `dialog_user` — reference | **один шард** | **локально** (reference-копия на каждом узле) | Вся переписка на одном шарде, минимум изменений схемы, запросы не меняются |
| `chat_key` (пара пользователей) | добавить `chat_key = hash(sorted(a,b))` в `dialog` и `dialog_message`, co-locate | один шард | один шард | Лучшая локальность, но нужна новая колонка, бэкофилл и переписывание `GetDialogID` |
| `from` (отправитель) | `dialog_message` по `from` | **scatter-gather по всем шардам** (сообщения диалога разъезжаются по 2 шардам) | — | Отклонён: ломает локальность чтения переписки |

**Выбрано: `dialog_id` + reference-таблицы.** Обоснование:
- `create_reference_table(t)` создаёт один шард, реплицируемый на **каждый**
  worker → join'ы и внешние ключи к нему выполняются локально, без сети;
- маленькие `users` / `dialog` / `dialog_user` дёшево держать полными копиями;
- распределённая таблица может ссылаться внешним ключом на reference-таблицу
  (`dialog_message.dialog_id → dialog`, `from/to → users`) — поэтому `users`
  тоже делаем reference (reference-таблица не может ссылаться на обычную
  локальную таблицу);
- Citus требует, чтобы distribution column входил в каждый PK/UNIQUE, поэтому
  PK `dialog_message` меняем с `(id)` на `(dialog_id, id)`. Этот же составной PK
  служит replica identity, необходимым для **неблокирующего** (через логическую
  репликацию) перемещения шардов при решардинге.

## Архитектура стека

```
                 ┌────────────────────┐
   приложение ── │  citus-coordinator │  (metadata + маршрутизация)
   (sm-app)      └─────────┬──────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
        ┌───────────┐ ┌───────────┐ ┌───────────┐
        │  worker-1 │ │  worker-2 │ │  worker-3 │  ← добавляется при решардинге
        └───────────┘ └───────────┘ └───────────┘
  distributed: dialog_message  (32 шарда, hash по dialog_id)
  reference:   users, dialog, dialog_user  (полная копия на каждом worker)
```

Файлы:
- `deployments/docker-compose-citus.yaml` — координатор + 2 воркера (+ 3-й под
  профилем `reshard`) + one-shot `citus-init` + `sm-app`;
- `deployments/citus/init.sh` — регистрирует воркеры, ждёт схему, применяет SQL;
- `deployments/citus/distribute.sql` — reference/distributed таблицы;
- `configs/app-config-citus.yaml` — приложение смотрит на координатор, без реплик.

## Шаг 1. Поднять кластер

Из корня репозитория:

```bash
docker compose --env-file .env -f deployments/docker-compose-citus.yaml up --build
```

Что происходит по шагам:
1. поднимаются `citus-coordinator`, `citus-worker-1`, `citus-worker-2`;
2. `sm-app` дожидается healthy-координатора и применяет goose-миграции (создаёт
   обычные таблицы на координаторе);
3. `citus-init` регистрирует воркеры (`citus_add_node`), дожидается появления
   таблицы `dialog_message` и применяет `distribute.sql`, который делает
   `users`/`dialog`/`dialog_user` reference-таблицами и распределяет
   `dialog_message` по `dialog_id` на 32 шарда.

## Шаг 2. Проверить распределение

```bash
docker compose --env-file .env -f deployments/docker-compose-citus.yaml \
  exec citus-coordinator psql -U "$DB_USER" -d "$DB_NAME" -c \
  "SELECT nodename, nodeport, isactive FROM pg_dist_node;"
#   → citus-worker-1, citus-worker-2

docker compose --env-file .env -f deployments/docker-compose-citus.yaml \
  exec citus-coordinator psql -U "$DB_USER" -d "$DB_NAME" -c \
  "SELECT nodename, count(*) AS shards
     FROM citus_shards
    WHERE table_name='dialog_message'::regclass
    GROUP BY nodename;"
#   → 32 шарда, поровну между двумя воркерами (по 16)

docker compose ... exec citus-coordinator psql -U "$DB_USER" -d "$DB_NAME" -c \
  "SELECT logicalrelid, partmethod FROM pg_dist_partition;"
#   → dialog_message = 'h' (hash);  reference-таблицы = 'n'
```

Отправьте пару сообщений через API (`POST /api/v1/dialog/{user_id}/send`) и
убедитесь, что чтение (`GET /api/v1/dialog/{user_id}/list`) работает как раньше —
координатор сам маршрутизирует запрос на нужный шард.

## Решардинг

Задача: добавить мощность (новый worker) и перераспределить шарды **без остановки
сервиса**. Citus перемещает шарды через **логическую репликацию PostgreSQL**,
поэтому чтения и записи продолжают идти во время перемещения.

**1. Запустить третий worker:**
```bash
docker compose --env-file .env -f deployments/docker-compose-citus.yaml \
  --profile reshard up -d citus-worker-3
```

**2. Зарегистрировать его на координаторе:**
```sql
SELECT citus_add_node('citus-worker-3', 5432);
```

**3. Посмотреть план перемещения (без выполнения):**
```sql
SELECT * FROM get_rebalance_table_shards_plan('dialog_message');
```

**4. Запустить онлайн-ребаланс (неблокирующий):**
```sql
SELECT citus_rebalance_start(shard_transfer_mode => 'force_logical');
```

**5. Следить за прогрессом, одновременно нагружая API:**
```sql
SELECT * FROM citus_rebalance_status();
SELECT * FROM get_rebalance_progress();
```
Держите в соседнем терминале цикл `curl` по `send` + `list` — он не должен вернуть
ни одной ошибки за время ребаланса. Это и есть доказательство отсутствия
даунтайма.

**6. Reference-таблицы** копируются на новый узел автоматически при ребалансе;
при необходимости можно форсировать:
```sql
SELECT replicate_reference_tables();
```

**7. Проверить результат** — 32 шарда теперь распределены по трём воркерам
(≈11 на каждом):
```sql
SELECT nodename, count(*) FROM citus_shards
 WHERE table_name='dialog_message'::regclass GROUP BY nodename;
```

### (Опционально) Сплит «горячего» шарда

Если один шард стал слишком большим/нагруженным, его можно разбить онлайн:
```sql
SELECT citus_split_shard_by_split_points(
  <shard_id>,
  ARRAY['<hash_point>'],      -- точка(и) разреза в пространстве hash
  ARRAY[<node_id_1>, <node_id_2>],
  'force_logical');
```

## Откат

Вернуть таблицу к обычному (нераспределённому) виду на координаторе:
```sql
SELECT undistribute_table('dialog_message');
SELECT undistribute_table('dialog_user');
SELECT undistribute_table('dialog');
SELECT undistribute_table('users');
```

## Замечания и риски

- Образ `citusdata/citus:13.0` собран на конкретной мажорной версии PostgreSQL —
  проверьте её (`SELECT version();`) и при необходимости зафиксируйте тег,
  совместимый с остальным стеком; тома данных несовместимы между мажорными
  версиями PG.
- Аутентификация в кластере — `trust` внутри приватной сети `pgnet`. Это допустимо
  только для учебного/dev-окружения; в проде используйте пароль/scram и
  ограничьте `pg_hba.conf`.
- Неблокирующее перемещение шардов требует replica identity — здесь оно
  обеспечено составным PK `(dialog_id, id)`.
- Если при `create_reference_table('users')` Citus ругается на внешние ключи от
  ещё локальных таблиц, убедитесь, что первым выполнен
  `citus_set_coordinator_host(...)` (он есть в `distribute.sql`) — это переводит
  локальные таблицы координатора в режим «Citus local» и разрешает FK на
  reference-таблицы.
