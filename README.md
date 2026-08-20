# test-api

REST API для задач внутри команд: роли, история изменений, комментарии,
кэш списка задач в Redis и SQL-отчёт по команде.

Стек: Go 1.25, MySQL 8, Redis 7, Docker Compose, OpenAPI 3.

## Быстрый старт

```bash
cp .env.example .env
# лучше заменить JWT_SECRET:
#   openssl rand -hex 32
docker compose up -d --build
```

Сервис слушает `http://localhost:8080`. Миграции накатываются при старте (`MYSQL_AUTO_MIGRATE=true`).

Остановить и стереть данные:

```bash
docker compose down -v
```

Локальная разработка без контейнера API — поднять только базы и запустить бинарь с хоста:

```bash
docker compose up -d mysql redis
go run ./cmd/api
```

## Конфигурация

Обязательны `MYSQL_DSN` и `JWT_SECRET` (≥ 32 символа).
Без них процесс не стартует.

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | адрес прослушивания |
| `MYSQL_DSN` | — (обязательна) | DSN, `app:app@tcp(mysql:3306)/test_api` |
| `MYSQL_AUTO_MIGRATE` | `true` | применять миграции при старте |
| `REDIS_ADDR` | `127.0.0.1:6379` | адрес Redis |
| `JWT_SECRET` | — (обязательна, ≥ 32) | ключ подписи JWT |
| `JWT_TTL` | `24h` | жизнь токена |
| `CACHE_TASK_LIST_TTL` | `5m` | TTL кеша списка задач |

В Compose хост MySQL — `mysql`, Redis — `redis` (имена сервисов).
`MYSQL_DSN` и `REDIS_ADDR` из `.env` нужны только для `go run` на хосте.

## Миграции

SQL лежит в `migrations/` и вшит в бинарник через `embed`. Контейнеру внешние файлы не нужны.
Повторный старт безопасен: golang-migrate берёт блокировку, `ErrNoChange` — не ошибка.

## Структура

```
cmd/api            точка входа: конфиг, зависимости, graceful shutdown
internal/domain    модели и правила (без импортов проекта)
internal/service   сценарии: права, транзакции, кэш
internal/store     SQL
internal/httpapi   роутер, middleware, хендлеры
internal/auth      JWT и bcrypt
internal/cache     Redis
internal/config    окружение
api/openapi.yaml   контракт
migrations/        схема БД
```

## Роли

| Роль | Права |
| --- | --- |
| `owner` | всё в своей команде, включая смену ролей и отчёт |
| `admin` | приглашать, править любые задачи, смотреть отчёт; роль создателя не меняет |
| `member` | видит команду, задачи, историю, комментарии; создаёт задачи; правит только свои |

`owner` выдаётся только при создании команды. Invite и PATCH дают только `admin` или `member`.

Чужая команда или задача → `404`, не `403`. Своя, но мало прав → `403`.

## API

Кроме `/register` и `/login` нужен заголовок `Authorization: Bearer <token>`.

| Метод | Путь | Описание |
| --- | --- | --- |
| `POST` | `/api/v1/register` | регистрация |
| `POST` | `/api/v1/login` | вход, JWT |
| `POST` | `/api/v1/teams` | создать команду |
| `GET` | `/api/v1/teams` | свои команды |
| `POST` | `/api/v1/teams/{team_id}/invite` | добавить участника |
| `PATCH` | `/api/v1/teams/{team_id}/members/{user_id}` | сменить роль |
| `GET` | `/api/v1/teams/{team_id}/stats` | SQL-отчёт |
| `POST` | `/api/v1/tasks` | создать задачу |
| `GET` | `/api/v1/tasks` | список задач |
| `PUT` | `/api/v1/tasks/{task_id}` | обновить задачу |
| `GET` | `/api/v1/tasks/{task_id}/history` | история |
| `POST` | `/api/v1/tasks/{task_id}/comments` | комментарий |
| `GET` | `/api/v1/tasks/{task_id}/comments` | комментарии |

Модели и коды ошибок — в `api/openapi.yaml`. 

### Примеры

```bash
API=http://localhost:8080/api/v1

curl -s -X POST $API/register -H 'Content-Type: application/json' \
  -d '{"email":"ann@example.com","name":"Ann","password":"sup3rsecret"}'

curl -s -X POST $API/register -H 'Content-Type: application/json' \
  -d '{"email":"bob@example.com","name":"Bob","password":"sup3rsecret"}'

TOKEN=$(curl -s -X POST $API/login -H 'Content-Type: application/json' \
  -d '{"email":"ann@example.com","password":"sup3rsecret"}' | jq -r .token)
AUTH="Authorization: Bearer $TOKEN"

curl -s -X POST $API/teams -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"name":"Platform"}'

curl -s $API/teams -H "$AUTH"

curl -s -X POST $API/teams/1/invite -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"email":"bob@example.com","role":"member"}'

curl -s -X PATCH $API/teams/1/members/2 -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"role":"admin"}'

curl -s -X POST $API/tasks -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"team_id":1,"title":"Ship the report","description":"one query","assignee_id":2}'

curl -s -G $API/tasks -H "$AUTH" \
  -d team_id=1 -d status=todo -d assignee_id=2 -d limit=20 -d offset=0

curl -s -X PUT $API/tasks/1 -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"status":"in_progress","version":1}'

curl -s -X PUT $API/tasks/1 -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"assignee_id":null}'

curl -s $API/tasks/1/history -H "$AUTH"

curl -s -X POST $API/tasks/1/comments -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"content":"Взял в работу"}'

curl -s $API/tasks/1/comments -H "$AUTH"

curl -s $API/teams/1/stats -H "$AUTH"
```

Ошибки единого вида:

```json
{ "error": { "code": "forbidden", "message": "..." } }
```