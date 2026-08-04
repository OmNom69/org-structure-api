# Org Structure API

Backend-сервис на Go для управления организационной структурой компании: иерархией подразделений и сотрудниками.

API поддерживает работу с древовидной структурой отделов, частичное обновление данных, безопасное изменение иерархии и несколько сценариев удаления подразделений. Бизнес-логика отделена от HTTP-слоя, доступ к данным скрыт за repository-интерфейсами, а инфраструктурные сценарии — логирование, healthcheck и корректное завершение приложения — реализованы отдельно.

При проектировании основное внимание уделено разделению ответственности, целостности данных, предсказуемому API-контракту и тестируемости бизнес-логики.

## Стек

- Go
- `net/http`
- PostgreSQL
- GORM
- Goose
- Docker / Docker Compose
- `log/slog`
- `testing`

## Инженерные решения

- слоистая архитектура `handler → service → repository`;
- dependency injection через конструкторы;
- repository-интерфейсы и тестовые реализации для unit-тестов;
- транзакции при сложных операциях с данными;
- единый JSON-формат ошибок;
- структурированное логирование и HTTP middleware;
- healthcheck приложения и PostgreSQL;
- graceful shutdown с использованием `context`, goroutine, channel и `select`.

## Архитектура

```text
HTTP client
    ↓
logging middleware
    ↓
handler
    ↓
service
    ↓
repository interface
    ↓
repository implementation
    ↓
GORM
    ↓
PostgreSQL
```

### Роль слоёв

| Слой | Роль |
| --- | --- |
| `cmd/app` | Создание зависимостей, регистрация маршрутов, запуск и завершение HTTP-сервера |
| `handler` | Разбор path/query/body, вызов service-слоя, формирование HTTP-ответов |
| `service` | Бизнес-логика, валидация и правила предметной области |
| `repository` | Работа с PostgreSQL через GORM |
| `model` | Структуры, соответствующие таблицам базы данных |
| `dto` | Форматы сложных ответов API |
| `validator` | Общая валидация входных данных |
| `middleware` | Сквозное логирование HTTP-запросов |
| `storage` | Ошибки уровня хранения данных |
| `config` | Загрузка конфигурации из переменных окружения |
| `database` | Создание подключения к PostgreSQL |

Зависимости создаются в `cmd/app/main.go` и передаются через конструкторы:

```text
repository → service → handler
```

Service-слой зависит от repository-интерфейсов, а не от конкретной реализации. Благодаря этому бизнес-логику можно тестировать с тестовыми реализациями репозиториев без запуска PostgreSQL.

## Возможности

### Подразделения

- создание корневых и дочерних подразделений;
- получение дерева подразделений;
- ограничение глубины дерева через `depth`;
- включение или исключение сотрудников из дерева через `include_employees`;
- переименование подразделения;
- перенос подразделения в другой родительский отдел;
- перенос подразделения в корень через `parent_id: null`;
- проверка существования родительского подразделения;
- уникальность имени подразделения внутри одного родителя;
- защита от циклов при изменении иерархии;
- удаление подразделения в режиме `cascade`;
- удаление подразделения в режиме `reassign`;
- транзакционный перенос сотрудников и дочерних подразделений при `reassign`.

### Сотрудники

- создание сотрудника внутри подразделения;
- получение списка сотрудников;
- получение сотрудника по `id`;
- изменение имени и должности;
- изменение даты найма;
- очистка даты найма через `hired_at: null`;
- перенос сотрудника в другое подразделение через `department_id`;
- проверка существования целевого подразделения;
- удаление сотрудника.

### Инфраструктура API

- healthcheck приложения и PostgreSQL;
- структурированное логирование через `log/slog`;
- middleware с логированием метода, пути, статуса и длительности запроса;
- единый JSON-формат ошибок для ошибок входных данных и service-слоя;
- преобразование доменных ошибок в HTTP-статусы и стабильные error codes;
- скрытие деталей внутренних ошибок от клиента;
- graceful shutdown при `Ctrl+C` и `SIGTERM`;
- таймаут чтения HTTP-заголовков;
- контейнеризация приложения и PostgreSQL;
- версионирование схемы БД через Goose;
- unit-тесты части `EmployeeService`.

## Детали реализации

### Service layer

HTTP-обработчики не содержат бизнес-логику и не обращаются к базе данных напрямую.

Handler отвечает за HTTP:

```text
path / query / body
        ↓
формирование input
        ↓
вызов service
        ↓
HTTP status + JSON response
```

Service отвечает за правила предметной области:

```text
валидация
проверка существования сущностей
контроль уникальности
защита дерева от циклов
правила PATCH
правила cascade / reassign
маппинг ошибок repository-слоя
```

Такое разделение упрощает сопровождение кода и позволяет тестировать бизнес-логику отдельно от транспорта и базы данных.

### DTO для дерева подразделений

Дерево подразделений не является прямой моделью таблицы БД. Ответ собирается в отдельный DTO:

```text
model.Department
        ↓
service.buildDepartmentTree
        ↓
dto.DepartmentTreeResponse
        ↓
JSON
```

Модели хранения данных не смешиваются с форматом сложного API-ответа.

### Три состояния PATCH-полей

Для некоторых полей API различает три состояния:

```text
поле не передано   → значение не изменяется
поле = null        → значение очищается
поле = значение    → значение обновляется
```

Для подразделений это используется у `parent_id`:

```json
{
  "parent_id": null
}
```

Такой запрос переносит подразделение в корень.

Для сотрудников это используется у `hired_at`:

```json
{
  "hired_at": null
}
```

Такой запрос очищает дату найма.

### Защита дерева от циклов

При изменении родителя сервис проверяет, что подразделение не перемещается внутрь собственного поддерева.

Исходная структура:

```text
IT
└── Backend
    └── Go Team
```

Операция, при которой `IT` становится дочерним подразделением `Go Team`, отклоняется: она создала бы цикл и нарушила целостность дерева.

### Два режима удаления подразделения

#### Cascade

```http
DELETE /departments/{id}?mode=cascade
```

Подразделение удаляется вместе с дочерними подразделениями и сотрудниками. Связи в PostgreSQL используют `ON DELETE CASCADE`.

#### Reassign

```http
DELETE /departments/{id}?mode=reassign&reassign_to_department_id=1
```

Перед удалением:

1. прямые дочерние подразделения переносятся в целевое подразделение;
2. сотрудники переносятся в целевое подразделение;
3. исходное подразделение удаляется.

Операция выполняется в транзакции. Если один из шагов завершается ошибкой, изменения откатываются.

### Централизованная обработка ошибок

Ошибки service-слоя преобразуются в:

- HTTP-статус;
- стабильный строковый `code`;
- безопасное сообщение для клиента.

Неизвестные внутренние ошибки полностью записываются в логи, но клиент получает обобщённый ответ без деталей базы данных.

## Запуск через Docker

### Требования

Перед запуском должны быть установлены:

- Docker;
- Docker Compose;
- Goose — для применения миграций с компьютера.

### 1. Клонировать репозиторий

```bash
git clone https://github.com/OmNom69/org-structure-api.git
cd org-structure-api
```

### 2. Создать `.env.docker`

Linux/macOS:

```bash
cp .env.docker.example .env.docker
```

PowerShell:

```powershell
Copy-Item .env.docker.example .env.docker
```

Пример:

```env
APP_PORT=8080

DB_HOST=db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=org_structure
DB_SSLMODE=disable

POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_password
POSTGRES_DB=org_structure
```

### 3. Запустить приложение и PostgreSQL

```bash
docker compose up --build -d
```

Проверить состояние контейнеров:

```bash
docker compose ps 
```

После запуска:

| Сервис | Адрес с компьютера |
| --- | --- |
| API | `http://localhost:8081` |
| PostgreSQL | `localhost:5433` |

Проброс портов:

```text
localhost:8081 → app:8080
localhost:5433 → db:5432
```

### 4. Применить миграции

Goose запускается с компьютера, поэтому используется `localhost:5433`:

```bash
goose -dir migrations postgres "host=localhost port=5433 user=postgres password=your_password dbname=org_structure sslmode=disable" up
```

Проверить статус миграций:

```bash
goose -dir migrations postgres "host=localhost port=5433 user=postgres password=your_password dbname=org_structure sslmode=disable" status
```

Откатить последнюю миграцию:

```bash
goose -dir migrations postgres "host=localhost port=5433 user=postgres password=your_password dbname=org_structure sslmode=disable" down
```

### 5. Проверить healthcheck

```http
GET http://localhost:8081/health
```

Успешный ответ:

```json
{
  "status": "ok",
  "database": "available"
}
```

Если PostgreSQL недоступен, API возвращает `503 Service Unavailable`:

```json
{
  "status": "unhealthy",
  "database": "unavailable"
}
```

## Переменные окружения

| Переменная | Назначение | Значение по умолчанию |
| --- | --- | --- |
| `APP_PORT` | Порт HTTP-сервера внутри контейнера | `8080` |
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_PORT` | Порт PostgreSQL | `5432` |
| `DB_USER` | Пользователь БД | `postgres` |
| `DB_PASSWORD` | Пароль БД | `postgres` |
| `DB_NAME` | Название БД | `org_structure` |
| `DB_SSLMODE` | Режим SSL | `disable` |

В `.env.docker` также используются `POSTGRES_USER`, `POSTGRES_PASSWORD` и `POSTGRES_DB` для инициализации контейнера PostgreSQL.

## API

Базовый адрес при запуске через Docker:

```text
http://localhost:8081
```

### Краткая таблица endpoints

| Метод | Endpoint | Назначение |
| --- | --- | --- |
| `GET` | `/health` | Проверить приложение и PostgreSQL |
| `POST` | `/departments/` | Создать подразделение |
| `GET` | `/departments/{id}` | Получить дерево подразделения |
| `PATCH` | `/departments/{id}` | Обновить или перенести подразделение |
| `DELETE` | `/departments/{id}` | Удалить подразделение |
| `POST` | `/departments/{id}/employees/` | Создать сотрудника в подразделении |
| `GET` | `/employees/` | Получить всех сотрудников |
| `GET` | `/employees/{id}` | Получить сотрудника |
| `PATCH` | `/employees/{id}` | Обновить сотрудника |
| `DELETE` | `/employees/{id}` | Удалить сотрудника |

### Healthcheck

```http
GET /health
```

Проверяет работу HTTP-приложения и выполняет `PingContext` к PostgreSQL.

### Создать подразделение

```http
POST /departments/
```

Корневое подразделение:

```json
{
  "name": "IT"
}
```

Дочернее подразделение:

```json
{
  "name": "Backend",
  "parent_id": 1
}
```

Если `parent_id` отсутствует или равен `null`, подразделение создаётся в корне.

Успешный статус: `201 Created`.

### Получить дерево подразделения

```http
GET /departments/{id}
```

Query-параметры:

| Параметр | Описание | По умолчанию |
| --- | --- | --- |
| `depth` | Глубина дерева от `1` до `5` | `1` |
| `include_employees` | Включать сотрудников в ответ | `true` |

Пример:

```http
GET /departments/1?depth=3&include_employees=true
```

Пример структуры ответа:

```json
{
  "id": 1,
  "name": "IT",
  "parent_id": null,
  "created_at": "2026-08-04T12:00:00Z",
  "employees": [],
  "children": [
    {
      "id": 2,
      "name": "Backend",
      "parent_id": 1,
      "created_at": "2026-08-04T12:05:00Z",
      "employees": [],
      "children": []
    }
  ]
}
```

Пример дерева:

```text
IT
├── Backend
│   ├── Go Team
│   └── Java Team
└── Frontend
    └── Web Team
```

### Обновить подразделение

```http
PATCH /departments/{id}
```

Переименовать:

```json
{
  "name": "Platform Team"
}
```

Перенести в другое подразделение:

```json
{
  "parent_id": 2
}
```

Перенести в корень:

```json
{
  "parent_id": null
}
```

Изменить имя и родителя одновременно:

```json
{
  "name": "Core Platform",
  "parent_id": 2
}
```

Пустой объект `{}` отклоняется.

### Удалить подразделение каскадом

```http
DELETE /departments/{id}?mode=cascade
```

Удаляет подразделение, его дочерние подразделения и сотрудников.

Пример ответа:

```json
{
  "message": "department deleted",
  "id": 4,
  "mode": "cascade"
}
```

### Удалить подразделение с переносом содержимого

```http
DELETE /departments/{id}?mode=reassign&reassign_to_department_id=1
```

Переносит прямые дочерние подразделения и сотрудников в указанное подразделение, затем удаляет исходное.

Пример ответа:

```json
{
  "message": "department deleted",
  "id": 4,
  "mode": "reassign",
  "reassign_to_department_id": 1
}
```

### Создать сотрудника

```http
POST /departments/{id}/employees/
```

```json
{
  "full_name": "Ivan Ivanov",
  "position": "Backend Developer",
  "hired_at": "2026-06-14"
}
```

`hired_at` необязателен и должен использовать формат `YYYY-MM-DD`.

Успешный статус: `201 Created`.

### Получить всех сотрудников

```http
GET /employees/
```

Пример ответа:

```json
[
  {
    "id": 1,
    "department_id": 2,
    "full_name": "Ivan Ivanov",
    "position": "Backend Developer",
    "hired_at": "2026-06-14T00:00:00Z",
    "created_at": "2026-06-14T12:00:00Z"
  }
]
```

### Получить сотрудника

```http
GET /employees/{id}
```

Пример ответа:

```json
{
  "id": 1,
  "department_id": 2,
  "full_name": "Ivan Ivanov",
  "position": "Backend Developer",
  "hired_at": "2026-06-14T00:00:00Z",
  "created_at": "2026-06-14T12:00:00Z"
}
```

### Обновить сотрудника

```http
PATCH /employees/{id}
```

Можно передавать только изменяемые поля.

Обновление нескольких полей:

```json
{
  "full_name": "Ivan Petrov",
  "position": "Senior Backend Developer",
  "department_id": 3,
  "hired_at": "2026-07-01"
}
```

Перенос в другое подразделение:

```json
{
  "department_id": 3
}
```

Очистка даты найма:

```json
{
  "hired_at": null
}
```

Пустой объект `{}` отклоняется. Поля `full_name`, `position` и `department_id` нельзя передавать как `null`.

### Удалить сотрудника

```http
DELETE /employees/{id}
```

Пример ответа:

```json
{
  "message": "employee deleted",
  "id": 1
}
```

## Формат ошибок

Ошибки API возвращаются в едином JSON-формате:

```json
{
  "error": {
    "code": "department_not_found",
    "message": "department not found"
  }
}
```

### Примеры error codes

| HTTP-статус | `code` | Значение |
| --- | --- | --- |
| `400` | `invalid_request_body` | Некорректный JSON body |
| `400` | `invalid_department_id` | Некорректный ID подразделения |
| `400` | `invalid_employee_id` | Некорректный ID сотрудника |
| `400` | `invalid_depth` | Глубина дерева вне диапазона `1..5` |
| `400` | `invalid_hired_at` | Неверный формат даты найма |
| `400` | `nothing_to_update` | В PATCH не переданы изменяемые поля |
| `400` | `department_cycle_detected` | Перенос создаёт цикл |
| `404` | `department_not_found` | Подразделение не найдено |
| `404` | `employee_not_found` | Сотрудник не найден |
| `409` | `department_already_exists` | Имя уже занято внутри этого родителя |
| `500` | `internal_server_error` | Внутренняя ошибка сервера |

Подробности неизвестных внутренних ошибок записываются в логи, но не отправляются клиенту.

## Логирование

Приложение пишет структурированные JSON-логи через `log/slog`.

Для каждого HTTP-запроса middleware записывает:

- HTTP-метод;
- путь;
- статус ответа;
- длительность выполнения.

Пример:

```json
{
  "time": "2026-08-04T12:26:13Z",
  "level": "INFO",
  "msg": "http request completed",
  "method": "GET",
  "path": "/health",
  "status": 200,
  "duration": "692µs"
}
```

Уровень лога зависит от статуса:

```text
2xx–3xx → INFO
4xx     → WARN
5xx     → ERROR
```

Отдельно логируются:

- создание, изменение и удаление сущностей;
- внутренние ошибки service-слоя;
- ошибки healthcheck;
- запуск и остановка сервера.

Просмотреть последние логи:

```bash
docker compose logs app --tail 50
```

Следить за логами в реальном времени:

```bash
docker compose logs app -f
```

## Graceful shutdown

HTTP-сервер обрабатывает `Ctrl+C` и `SIGTERM`.

При остановке приложение:

1. получает сигнал завершения;
2. прекращает принимать новые запросы;
3. ждёт завершения активных запросов;
4. ограничивает ожидание десятью секундами;
5. при неудаче принудительно закрывает сервер.

Также настроен `ReadHeaderTimeout` в 5 секунд, чтобы клиент не мог бесконечно удерживать соединение, слишком медленно отправляя HTTP-заголовки.

Проверка через Docker:

```bash
docker compose stop app
docker compose logs app
```

Ожидаемые логи:

```text
shutdown signal received
server stopped gracefully
```

## Тесты

Запустить все тесты:

```bash
go test ./...
```

Сейчас unit-тестами покрыта часть `EmployeeService`:

- отклонение некорректного ID;
- успешное получение сотрудника;
- успешное получение списка сотрудников;
- успешное создание сотрудника;
- проверка взаимодействия service с тестовыми реализациями репозиториев.

Тесты не требуют запуска PostgreSQL, потому что service-слой зависит от repository-интерфейсов.

## Структура проекта

```text
org-structure-api/
├── cmd/
│   └── app/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── database.go
│   ├── dto/
│   │   └── department.go
│   ├── handler/
│   │   ├── department_handler.go
│   │   ├── employee_handler.go
│   │   ├── error_status.go
│   │   ├── health_handler.go
│   │   └── optional.go
│   ├── middleware/
│   │   └── logging.go
│   ├── model/
│   │   ├── department.go
│   │   └── employee.go
│   ├── repository/
│   │   ├── department_repository.go
│   │   └── employee_repository.go
│   ├── service/
│   │   ├── contracts.go
│   │   ├── department_service.go
│   │   ├── employee_service.go
│   │   ├── employee_service_test.go
│   │   └── errors.go
│   ├── storage/
│   │   └── errors.go
│   └── validator/
│       └── validation.go
├── migrations/
│   └── 01_create_departments_and_employees.sql
├── .dockerignore
├── .env.docker.example
├── .env.example
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## Следующие улучшения

- unit-тесты `DepartmentService`;
- HTTP-тесты handlers через `httptest`;
- пагинация, поиск и фильтрация сотрудников;
- индексы для `departments.parent_id` и `employees.department_id`;
- GitHub Actions для `go test`, `go vet` и race detector;
- OpenAPI / Swagger;
- integration-тесты repository-слоя с PostgreSQL;
- единый helper для кодирования успешных JSON-ответов;
- корректное закрытие пула соединений с PostgreSQL при завершении приложения.
