# Org Structure API

Backend API для управления организационной структурой компании.

Проект позволяет создавать отделы, строить дерево подразделений, добавлять сотрудников, обновлять и удалять отделы с разными режимами удаления.

## Стек

* Go
* net/http
* PostgreSQL
* GORM
* Goose migrations
* Docker / Docker Compose

## Что может делать

* Создание подразделений
* Создание сотрудников внутри подразделения
* Получение дерева подразделений
* Ограничение глубины дерева через `depth`
* Включение/выключение сотрудников в ответе через `include_employees`
* Обновление подразделения
* Удаление подразделения в режиме `cascade`
* Удаление подразделения в режиме `reassign`
* Валидация строковых полей
* Проверка существования родительского отдела
* Проверка уникальности имени отдела внутри одного родителя

## Запуск проекта

### 1. Клонировать репозиторий

```bash
git clone https://github.com/OmNom69/org-structure-api.git
cd org-structure-api
```

### 2. Создать `.env.docker`

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

`APP_PORT=8080` — это порт приложения внутри Docker-контейнера.

API будет доступно с компьютера по адресу:

```txt
http://localhost:8081
```

В `docker-compose.yml` используется проброс портов:

```yaml
ports:
  - "8081:8080" # API
  - "5433:5432" # PostgreSQL
```

### 3. Запустить проект через Docker Compose

```bash
docker compose up --build -d
```

### 4. Применить миграции

Миграции запускаются с компьютера, поэтому используется `host=localhost`:

```bash
goose -dir migrations postgres "host=localhost port=5433 user=postgres password=your_password dbname=org_structure sslmode=disable" up
```

После этого API доступно по адресу:

```txt
http://localhost:8081
```

## API endpoints

### Создать отдел

```http
POST /departments/
```

Body:

```json
{
  "name": "Backend",
  "parent_id": 1
}
```

Если `parent_id` не указан или равен `null`, отдел создаётся как корневой.

Пример корневого отдела:

```json
{
  "name": "IT"
}
```

### Получить отдел с деревом

```http
GET /departments/{id}
```

Query parameters:

| Параметр            | Описание                       | Значение по умолчанию |
| ------------------- | ------------------------------ | --------------------- |
| `depth`             | Глубина дерева от 1 до 5       | `1`                   |
| `include_employees` | Показывать сотрудников или нет | `true`                |

Пример:

```http
GET /departments/1?depth=3&include_employees=true
```

### Обновить отдел

```http
PATCH /departments/{id}
```

Body:

```json
{
  "name": "New Backend",
  "parent_id": 2
}
```

Можно передавать только одно поле:

```json
{
  "name": "Platform Team"
}
```

### Удалить отдел каскадом

```http
DELETE /departments/{id}?mode=cascade
```

Удаляет отдел, его дочерние отделы и сотрудников.

### Удалить отдел с переносом содержимого

```http
DELETE /departments/{id}?mode=reassign&reassign_to_department_id=1
```

Переносит прямые дочерние отделы и сотрудников в другой отдел, после чего удаляет исходный отдел.

### Создать сотрудника в отделе

```http
POST /departments/{id}/employees/
```

Body:

```json
{
  "full_name": "Ivan Ivanov",
  "position": "Backend Developer",
  "hired_at": "2026-06-14"
}
```

Поле `hired_at` необязательное.

## Пример дерева

```txt
IT
├── Backend
│   ├── Golang-Team
│   └── Java-Team
└── Frontend
    └── JS-Team
```

Запрос:

```http
GET /departments/1?depth=2&include_employees=true
```

вернёт отдел `IT`, его дочерние отделы и сотрудников, если они есть.

## Структура проекта

```txt
org-structure-api/
├── cmd/app/main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── handler/
│   ├── model/
│   ├── repository/
├── migrations/
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

## Что ещё можно улучшить

* Добавить service-слой
* Добавить защиту от циклов дерева
* Добавить транзакции для `reassign`
* Добавить тесты
* Улучшить логирование через `slog`
