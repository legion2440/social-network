# Social Network

Полноценная социальная сеть в стиле Facebook на Go, SQLite, WebSocket, собственном декларативном JavaScript-фреймворке и двухконтейнерном Docker-развёртывании.

Приложение поддерживает публичные и приватные профили, подписки, посты с контролем аудитории, медиа-вложения, группы, события, сохраняемые уведомления, личные и групповые чаты, а также счётчики непрочитанного.

[English version](README.md) · [Контракты backend](backend/README.md)

## 📋 Оглавление

- [🚀 Быстрый запуск](#-быстрый-запуск)
- [📝 О проекте](#-о-проекте)
- [✨ Возможности](#-возможности)
- [🏗️ Архитектура](#️-архитектура)
- [🧰 Стек технологий](#-стек-технологий)
- [🔐 Безопасность и контроль доступа](#-безопасность-и-контроль-доступа)
- [🐳 Docker-развёртывание](#-docker-развёртывание)
- [💾 Хранение данных и миграции](#-хранение-данных-и-миграции)
- [⚙️ Конфигурация](#️-конфигурация)
- [🧪 Локальная разработка и тесты](#-локальная-разработка-и-тесты)
- [📁 Структура проекта](#-структура-проекта)
- [📚 Документация API](#-документация-api)
- [🧹 Очистка](#-очистка)
- [⚠️ Примечания](#️-примечания)
- [🧑‍💻 Автор](#-автор)

## 🚀 Быстрый запуск

### Требования

- Docker Engine
- Docker Compose plugin
- свободный порт `8080` или другой порт через `SOCIAL_NETWORK_PORT`

### Запуск через Docker Compose

```bash
git clone https://github.com/legion2440/social-network.git
cd social-network

docker compose config
docker compose up --build -d
docker compose ps
```

Открыть:

```text
http://127.0.0.1:8080
```

Другой host-порт:

```bash
SOCIAL_NETWORK_PORT=8081 docker compose up --build -d
```

Остановить контейнеры, сохранив базу и загруженные файлы:

```bash
docker compose down --remove-orphans
```

## 📝 О проекте

Social Network является полноценным учебным веб-приложением, реализующим основную логику современной социальной платформы.

Backend отвечает за аутентификацию, приватность, контроль доступа, хранение данных, миграции, метаданные медиа, уведомления, счётчики непрочитанного и авторизацию WebSocket-операций. Frontend представляет собой статическое SPA, обслуживаемое Caddy, и работает с backend через same-origin HTTP и WebSocket-маршруты.

Backend намеренно остаётся приватным внутри Docker-сети. На host публикуется только frontend-контейнер.

## ✨ Возможности

### Аутентификация и профили

- регистрация с email, паролем, именем, фамилией и датой рождения;
- необязательные avatar, gender, nickname и about;
- HttpOnly session cookie и несколько независимых сессий для одного пользователя;
- login, восстановление сессии и logout с любой страницы приложения;
- редактирование профиля;
- замена и удаление avatar;
- публичный и приватный режимы профиля;
- скрытие полной информации приватного профиля от посторонних пользователей.

### Подписки

- мгновенная подписка на публичный профиль;
- follow request для приватного профиля;
- accept и decline входящей заявки;
- unfollow и отмена pending-заявки;
- списки followers и following с relationship-состоянием относительно текущего пользователя.

### Посты и комментарии

- личные посты с тремя режимами приватности:
  - `public`;
  - `followers`;
  - `selected`;
- выбор конкретных accepted followers для `selected`-поста;
- вложения JPEG, PNG, GIF и WebP до 20 MB;
- пагинация ленты и постов профиля;
- комментарии с необязательным JPEG, PNG, GIF или WebP;
- повторная проверка текущего доступа при каждом запросе защищённого медиа поста или комментария.

### Группы

- создание группы с названием и описанием;
- каталог групп и текущее membership-состояние;
- приглашения от владельца или любого активного участника;
- join requests, которые принимает или отклоняет владелец;
- принятие и отклонение приглашения;
- посты и комментарии только для участников группы;
- групповые события с названием, описанием, датой/временем и RSVP:
  - Going;
  - Not going.

### Чат в реальном времени

- личный чат, если хотя бы в одном направлении существует accepted follow;
- групповой чат только для активных участников;
- доставка сообщений через WebSocket без обновления страницы;
- сохраняемая история личных и групповых сообщений;
- отправка emoji;
- typing и presence state;
- сохраняемые счётчики непрочитанного для каждого чата;
- отзыв сессии закрывает связанный realtime-доступ.

### Уведомления

Сохраняемые уведомления создаются для обязательных событий:

- follow request к приватному профилю;
- приглашение в группу;
- запрос на вступление в группу;
- создание события в группе.

Поддерживаются:

- realtime-доставка;
- unread count;
- mark one as read;
- mark all as read;
- accept/decline для actionable-уведомлений;
- идемпотентная обработка lifecycle.

### Медиа

- определение JPEG, PNG, GIF и WebP по содержимому файла;
- лимит 20 MB;
- транзакционное создание метаданных и связи с объектом;
- удаление staged-файла при ошибке транзакции;
- контролируемые маршруты avatar, post и comment media;
- отсутствие общего публичного каталога uploads.

## 🏗️ Архитектура

```text
browser
  |
  v
social-network-frontend:8080
  Caddy, публичный контейнер
  |
  +--> static SPA
  |
  +--> /api, /ws, /static/avatars
         |
         v
       social-network-backend:8080
         Go, приватный контейнер
         |
         +--> SQLite
         +--> uploaded files
         +--> migrations
         +--> WebSocket hub
```

На host публикуется только frontend-порт.

Caddy проксирует точные и вложенные маршруты:

```text
/api
/api/*
/ws
/ws/*
/static/avatars
/static/avatars/*
```

Удалённые legacy-маршруты получают прямой `404` до SPA fallback:

```text
/uploads
/uploads/*
```

Остальные пути обслуживаются из `/srv`. Для client-side routing используется fallback на `index.html`.

## 🧰 Стек технологий

| Слой | Технология |
|---|---|
| Backend | Go `1.24.2` |
| База данных | SQLite через `mattn/go-sqlite3` |
| Миграции | `golang-migrate` |
| Аутентификация | server-side sessions, cookies, bcrypt |
| Realtime | Gorilla WebSocket |
| ID | Google UUID |
| Frontend | собственный декларативный `dc-runtime` |
| Rendering layer | React и ReactDOM |
| Static server / reverse proxy | Caddy |
| Контейнеры | Docker и Docker Compose |

Browser-приложение не организовано как обычное React component tree. Собственный framework в `frontend/js/runtime.js` обрабатывает шаблон `<x-dc>` из `frontend/index.html`, а React и ReactDOM используются как rendering layer.

## 🔐 Безопасность и контроль доступа

- пароли хранятся как bcrypt hashes;
- session token передаётся через HttpOnly, SameSite=Lax cookie;
- Docker backend не публикуется на host;
- данные приватного профиля доступны только владельцу и accepted followers;
- доступ к posts, comments, groups, events и chats проверяется на backend;
- доступ к медиа повторно проверяется по текущей политике родительского объекта;
- MIME определяется по содержимому, а не по расширению файла;
- защищённые media responses используют `X-Content-Type-Options: nosniff`;
- runtime-контейнеры работают от numeric non-root users;
- root filesystem запускается в read-only режиме;
- используется `no-new-privileges`;
- graceful shutdown закрывает приём новых realtime-операций, дожидается WebSocket-работы и останавливает HTTP server.

Локальная HTTP-конфигурация использует:

```text
SOCIAL_NETWORK_COOKIE_SECURE=false
```

При развёртывании за HTTPS значение должно быть `true`.

## 🐳 Docker-развёртывание

Проект собирает два образа:

```text
social-network-backend:local
social-network-frontend:local
```

Оба образа используют закреплённые digest официальных base images.

### Ручная сборка образов

Из корня репозитория:

```bash
docker image build -f backend/Dockerfile -t social-network-backend:local .
docker image build -f frontend/Dockerfile -t social-network-frontend:local .
```

### Standalone-запуск

Один раз создать приватную сеть и постоянные named volumes:

```bash
docker network create social-network
docker volume create social-network-db
docker volume create social-network-uploads
```

Запустить приватный backend:

```bash
docker container run -d \
  --name social-network-backend \
  --network social-network \
  --read-only \
  --tmpfs /tmp \
  --security-opt no-new-privileges \
  --mount type=volume,source=social-network-db,target=/data/db \
  --mount type=volume,source=social-network-uploads,target=/data/uploads \
  social-network-backend:local
```

Проверить health backend:

```bash
docker inspect --format='{{json .State.Health}}' social-network-backend
```

Запустить публичный frontend:

```bash
docker container run -d \
  --name social-network-frontend \
  --network social-network \
  --read-only \
  --tmpfs /tmp \
  --security-opt no-new-privileges \
  -e BACKEND_UPSTREAM=social-network-backend:8080 \
  -p "${SOCIAL_NETWORK_PORT:-8080}:8080" \
  social-network-frontend:local
```

Проверить контейнеры, порты, health и logs:

```bash
docker ps -a
docker port social-network-backend
docker port social-network-frontend
docker inspect --format='{{json .State.Health}}' social-network-frontend
docker logs social-network-backend
docker logs social-network-frontend
```

Остановить и удалить standalone-контейнеры без удаления данных:

```bash
docker stop -t 15 social-network-frontend social-network-backend
docker rm social-network-frontend social-network-backend
docker network rm social-network
```

## 💾 Хранение данных и миграции

Сейчас приложение использует `15` версионированных SQLite migrations.

Pending migrations применяются автоматически до начала прослушивания HTTP. Текущая версия и dirty state хранятся в `schema_migrations`.

Compose и standalone используют одни и те же явно заданные volumes:

```text
social-network-db
social-network-uploads
```

Перезапуск с теми же volumes сохраняет:

- пользователей;
- сессии;
- профили и подписки;
- посты и комментарии;
- группы и события;
- личные и групповые сообщения;
- unread state уведомлений и чатов;
- загруженные файлы.

Backend healthcheck вызывает:

```text
GET /api/health
```

Статус healthy устанавливается только при корректной работе HTTP server и SQLite.

Frontend healthcheck проверяет статическую выдачу и проксируемый backend health endpoint.

Проверка migration state при локальной разработке:

```bash
cd backend
sqlite3 var/social-network.db \
  "SELECT version, dirty FROM schema_migrations;"
```

Ожидаемая текущая версия:

```text
15 | 0
```

## ⚙️ Конфигурация

Backend environment variables:

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `SOCIAL_NETWORK_HTTP_ADDR` | `127.0.0.1:8080` | адрес backend |
| `SOCIAL_NETWORK_DB_PATH` | `var/social-network.db` | путь к SQLite |
| `SOCIAL_NETWORK_UPLOAD_DIR` | `var/uploads` | каталог uploaded files |
| `SOCIAL_NETWORK_FRONTEND_DIR` | `../frontend` | локальный static frontend |
| `SOCIAL_NETWORK_COOKIE_SECURE` | `false` | Secure-атрибут session cookie |

Настройка host-порта Compose:

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `SOCIAL_NETWORK_PORT` | `8080` | опубликованный frontend port |

Docker заменяет backend paths на постоянные `/data` mounts и отключает выдачу frontend через Go server.

## 🧪 Локальная разработка и тесты

### Backend

SQLite driver требует CGO и рабочий C compiler.

```bash
cd backend

go run ./cmd/server
```

При локальном запуске Go server по умолчанию выдаёт `../frontend`, поэтому browser requests и session cookie остаются same-origin.

Backend checks:

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

### Frontend

Frontend tests используют встроенный Node test runner и не требуют production dependency install:

```bash
cd frontend
npm test
```

Итоговая локальная проверка:

```text
Go tests:              passed
Go race tests:         passed
go vet:                passed
Frontend tests:        121/121
docker compose config: passed
```

## 📁 Структура проекта

```text
social-network/
├── backend/
│   ├── cmd/
│   │   ├── healthcheck/
│   │   └── server/
│   ├── internal/
│   │   ├── app/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── http/
│   │   ├── platform/
│   │   ├── realtime/
│   │   ├── repo/
│   │   │   └── sqlite/
│   │   │       └── migrations/
│   │   └── service/
│   ├── Dockerfile
│   └── README.md
├── frontend/
│   ├── assets/
│   ├── css/
│   ├── js/
│   ├── Caddyfile
│   ├── Dockerfile
│   ├── index.html
│   └── package.json
├── compose.yaml
├── README.md
└── README_RU.md
```

## 📚 Документация API

Подробный backend README описывает:

- контракты migrations;
- authentication и profile rules;
- controlled media delivery;
- follower state;
- post privacy;
- comments и attachments;
- groups и membership transitions;
- events и RSVP;
- persisted notifications;
- direct и group chats;
- WebSocket message contracts;
- HTTP status behavior.

Файл:

```text
backend/README.md
```

## 🧹 Очистка

Остановить Compose-контейнеры, сохранив данные:

```bash
docker compose down --remove-orphans
```

Полностью удалить базу и загруженные файлы проекта:

```bash
docker compose down -v --remove-orphans
```

Standalone cleanup:

```bash
docker volume rm social-network-db social-network-uploads
```

Не используйте глобальный `docker system prune` для очистки проекта.

## ⚠️ Примечания

- Нельзя одновременно запускать standalone и Compose backend. Они используют одну SQLite и одни upload volumes.
- Перед сменой режима нужно остановить и удалить предыдущий backend.
- Два backend-процесса не должны одновременно работать с одним SQLite-файлом.
- Текущий интерфейс оптимизирован в первую очередь под desktop layout.
- Полные контракты приложения и persistence находятся в `backend/README.md`.

## 🧑‍💻 Авторы
Nazar Yestayev (@nyestaye)
Nurgul Ilyassova (@nilyasso)
Sultan Yersultan (@syersult)
Teniz Serikbayev (@tteniz)
Aiman Zhumabayeva (@azhumaba)
Aiymgul Gabdullina (@agabdullin)
