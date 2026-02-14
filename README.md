# 404chan Backend

Анонимный имиджборд на Go + Gin + GORM + Redis.

## Технологии

- **Go 1.24.1** — язык программирования
- **Gin** — HTTP-фреймворк
- **GORM** — ORM для PostgreSQL (с AutoMigrate)
- **Redis** — кэширование и PubSub
- **Gorilla WebSocket** — WebSocket
- **Uber Zap** — структурированное логирование

## Быстрый старт

### Требования

- Go 1.24+
- PostgreSQL 14+
- Redis 7+

### Установка

```bash
# Копируем переменные окружения
cp .env.example .env

# Устанавливаем зависимости
go mod tidy

# Запуск (миграции и сиды выполняются автоматически)
air

# Или сборка и запуск
make build
make run
```

### Отдельные команды

```bash
make migrate   # Только миграции
make seed      # Только сиды
```

## Структура проекта

```
internal/
├── app/              # Бизнес-логика (по фичам)
│   ├── board/        # Доски
│   ├── thread/       # Треды
│   ├── message/      # Сообщения
│   ├── user/         # Пользователи
│   ├── session/      # Сессии
│   └── health/       # Health check
├── config/           # Конфигурация
├── db/               # Подключение PostgreSQL
│   └── seeder/      # Сиды базы данных
├── gateways/        # Внешние сервисы (WebSocket)
├── middleware/       # HTTP middleware (CORS, логирование)
├── providers/        # Redis provider
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

## Миграции

Используется **GORM AutoMigrate** — миграции выполняются автоматически при запуске приложения.

При изменении моделей просто обновите структуры в `model.go`, GORM автоматически применит изменения.

## Сиды

Сиды (начальные данные) создаются автоматически при запуске:

- Доски по умолчанию: `a` (Anime), `b` (Random), `c` (Cute), `mu` (Music), `prog` (Programming), `sci` (Science)
- Примерные треды в доске `b`

## API эндпоинты

### Health Check

```http
GET /health
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
GET    /api/threads/:id                 # Тред с сообщениями
```

### Messages

```http
POST   /api/threads/:id/messages        # Ответ в тред
```

## WebSocket

```http
ws://localhost:8080/ws
```

## Лицензия

MIT
