# 404chan Backend

Анонимный имиджборд на Go + Gin + GORM + Redis.

## Технологии

- **Go 1.24+** — язык программирования
- **Gin** — HTTP-фреймворк
- **GORM** — ORM для PostgreSQL (с AutoMigrate)
- **Redis** — кэширование и PubSub
- **Gorilla WebSocket** — WebSocket
- **Uber Zap** — структурированное логирование
- **MinIO** — хранение файлов

## Запуск

Основной способ запуска — общий dev-стек из родительского репозитория `404chan`:

```bash
git clone --recurse-submodules git@github.com:k1rvl07/404chan.git
cd 404chan
make init
make dev
```

Dev-стек поднимает PostgreSQL, Redis, MinIO, backend и frontend в Docker, а Zellij показывает логи и shell-панели.

### Автономный запуск backend

Требования: Go 1.24+, PostgreSQL 14+, Redis 7+, MinIO.

```bash
cp .env.example .env
# В .env укажите локальные адреса сервисов:
# DB_HOST=localhost
# REDIS_URL=redis://default:redis_password@localhost:6379/0
# MINIO_URL=localhost:9000

go mod download
swag init
make dev      # swag init + air
```

Или сборка и запуск:

```bash
make build
make run
```

`docs/` генерируется командой `swag init` и не хранится в git. Без этого шага сборка невозможна, так как `main.go` импортирует `backend/docs`.

### Prod

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

Prod-стек backend включает PostgreSQL, Redis, MinIO и сам backend. Для общего prod-стека с frontend используйте `docker-compose.prod.yml` в корне репозитория.

## Команды

```bash
make swag     # генерация Swagger-документации
make build    # сборка бинарника
make run      # сборка и запуск
make dev      # swag init + air
make lint     # go vet
make test     # go test
make clean    # удалить tmp
```

## Структура проекта

```
internal/
├── app/              # Бизнес-логика (по фичам)
│   ├── attachment/   # Вложения
│   ├── board/        # Доски
│   ├── cleanup/      # Очистка
│   ├── health/       # Health check
│   ├── message/      # Сообщения
│   ├── session/      # Сессии
│   ├── thread/       # Треды
│   ├── upload/       # Загрузка файлов
│   └── user/         # Пользователи
├── config/           # Конфигурация
├── db/               # Подключение PostgreSQL
│   └── seeder/       # Сиды базы данных
├── gateways/         # WebSocket
├── middleware/       # HTTP middleware (CORS, логирование)
├── providers/        # Redis и MinIO
├── router/           # Маршрутизация
└── utils/            # Утилиты
```

### Архитектура слоёв

Каждая фича следует паттерну:

```
app/feature/
├── model.go      # Структуры данных
├── repository.go # Работа с БД
├── service.go    # Бизнес-логика
├── handler.go    # HTTP-обработчики
└── route.go      # Маршруты
```

## Миграции и сиды

Используется **GORM AutoMigrate** — миграции выполняются автоматически при запуске приложения. Сиды также создаются при старте (доски `a`, `b`, `c`, `mu`, `prog`, `sci`).

## API

### Health Check

```http
GET /api/health
```

### Boards

```http
GET    /api/boards         # Список досок
POST   /api/boards         # Создать доску
GET    /api/boards/:slug   # Доска по slug
```

### Threads

```http
GET    /api/boards/:slug/threads       # Список тредов в доске
POST   /api/boards/:slug/threads       # Создать тред
GET    /api/threads/:id                # Тред с сообщениями
```

### Messages

```http
POST   /api/threads/:id/messages       # Ответ в тред
```

## Swagger

```http
http://localhost:8080/swagger/index.html
```

## WebSocket

```http
ws://localhost:8080/ws
```

## Лицензия

MIT
