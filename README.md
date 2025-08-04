# 404chan Backend

**Анонимная имиджборда на Go + Gin + WebSocket**

Бэкенд часть full-stack приложения, написанного с использованием **Go**, **Gin**, **GORM** и **Redis**.  
Проект реализует анонимный WebSocket-чат с архитектурой, соответствующей принципам **Clean Architecture**, и легко масштабируется.

---

## ⚙️ Технологии

- [Go (Golang)](https://go.dev/) — язык программирования
- [Gin](https://github.com/gin-gonic/gin) — HTTP-фреймворк
- [GORM](https://gorm.io/) — ORM для PostgreSQL
- [Redis](https://redis.io/) — кэш и PubSub
- [Gorilla WebSocket](https://github.com/gorilla/websocket) — WebSocket-библиотека
- [zap](https://github.com/uber-go/zap) — структурированный логгер
- [.env](https://github.com/joho/godotenv) — загрузка переменных окружения
- [Docker](https://www.docker.com/) + [Docker Compose](https://docs.docker.com/compose/) — контейнеризация

---

## 🚀 Возможности

- ✅ **WebSocket**
  - Подключение через `/ws`
  - Поддержка множественных клиентов
  - Централизованное управление сессиями (WebSocket Hub)
- ✅ **Health Check**
  - Эндпоинт `/health` проверяет соединение с PostgreSQL и Redis
- ✅ **Строгая архитектура**
  - handler → service → repository → model
- ✅ **Redis Provider**
  - Подключение и PubSub для масштабируемости
- ✅ **Логирование**
  - Структурированные JSON-логи через `zap`
- ✅ **Загрузка конфигурации из `.env`**
- ✅ **Docker-ready**  
  - Быстрый запуск и удобство разработки

---

## 📁 Структура

```bash
backend/
├── cmd/                  # Точка входа (main.go)
├── internal/             # Приватная логика приложения
│   ├── app/              # Бизнес-логика (по фичам)
│   │   ├── health/       # Health Check
│   │   └── bootstrap.go  # Инициализация приложения 
│   ├── config/           # Загрузка конфигурации
│   ├── db/               # Инициализация PostgreSQL
│   ├── providers/        # Redis, WebSocketHub и др.
│   ├── gateways/         # Внешние зависимости (если есть)
│   ├── router/           # Маршруты HTTP + WebSocket
│   ├── middleware/       # CORS, логирование
│   └── utils/            # Хелперы, валидация
├── pkg/                  # Общие пакеты, ошибки
├── migrations/           # SQL-миграции
├── .env                  # Переменные окружения
├── Makefile              # Makefile
├── modd.conf             # Конфиг modd
├── go.mod / go.sum       # Зависимости Go
└── README.md             # Этот файл
```

---

## 🛠 Локальный запуск

> Требуется установленный Go, PostgreSQL, Redis

1. Скопируйте `.env.example` → `.env` и настройте
2. Установите зависимости:

```bash
go mod tidy
```

3. Запустите миграции (вручную или через `goose`)
4. Запустите приложение:

```bash
go run ./cmd/main.go
```

или с автоперезапуском (если установлен [modd](https://github.com/cortesi/modd)):

```bash
air
```

---

## 📡 WebSocket

- URL подключения: `ws://localhost:8080/ws`
- Каждому клиенту присваивается уникальный ID

---

## 🔍 Health Check

```http
GET /health
```

Ответ:

```json
{
  "DB": "ok",
  "Redis": "ok"
}
```

---

## 📄 Лицензия

Проект распространяется под лицензией [MIT](https://opensource.org/licenses/MIT)

---

## 🧠 Автор

Разработано как часть full-stack проекта [404chan](https://github.com/k1rvl07)
