# Social Network Frontend

Статический SPA frontend для Social Network.

Browser-приложение использует собственный декларативный JavaScript framework `dc-runtime`. React и ReactDOM являются его rendering layer, а не стандартной component architecture приложения.

Этот файл описывает frontend implementation, state, HTTP/WebSocket integration, защиту от stale responses, lifecycle media preview и тесты. Обзор продукта и Docker quick start находятся в [корневом README](../README_RU.md). Backend API и authorization contracts описаны в [backend README](../backend/README_RU.md).

[English version](README.md) · [Документация backend](../backend/README_RU.md)

## 📋 Оглавление

- [🚀 Запуск и тесты](#-запуск-и-тесты)
- [🧩 Модель выполнения](#-модель-выполнения)
- [⚙️ dc-runtime](#️-dc-runtime)
- [📄 Синтаксис template](#-синтаксис-template)
- [📁 Ответственность файлов](#-ответственность-файлов)
- [🗃️ Application state](#️-application-state)
- [🌐 HTTP client](#-http-client)
- [🔄 Нормализация и authoritative state](#-нормализация-и-authoritative-state)
- [🛡️ Request generations](#️-request-generations)
- [🔑 Authentication lifecycle](#-authentication-lifecycle)
- [📝 Posts, comments и media](#-posts-comments-и-media)
- [👥 Groups и events](#-groups-и-events)
- [🔔 Notifications](#-notifications)
- [💬 Chat и realtime](#-chat-и-realtime)
- [🧹 Очистка ресурсов](#-очистка-ресурсов)
- [📦 Production delivery](#-production-delivery)
- [🧪 Test suite](#-test-suite)
- [📁 Структура](#-структура)

## 🚀 Запуск и тесты

Frontend статический. Его обслуживает:

- Go backend при local development;
- Caddy в production Docker topology.

Локальный full-stack запуск из `backend/`:

```bash
cd ../backend
go run ./cmd/server
```

Открыть:

```text
http://127.0.0.1:8080
```

Frontend tests из `frontend/`:

```bash
npm test
```

Используется встроенный Node test runner. В `package.json` нет production dependencies и frontend build step.

## 🧩 Модель выполнения

Browser загружает приложение из `index.html`.

```text
index.html
  |
  +--> resources and model scripts
  +--> bundled React
  +--> bundled ReactDOM
  +--> AuthAPI
  +--> dc-runtime
  +--> application logic
```

`<x-dc>` содержит application template. `dc-runtime` парсит его, заменяет `<x-dc>` на `#dc-root`, компилирует directives и interpolations, создаёт React elements и монтирует их через ReactDOM.

UI не разбит на обычные `.jsx` React components. Основное поведение находится в одном `Component extends DCLogic`, state, methods и computed values которого доступны декларативному template.

## ⚙️ dc-runtime

`frontend/js/runtime.js` является generated runtime bundle. Его не следует редактировать вручную без rebuild source.

Runtime предоставляет:

- поиск `<x-dc>`;
- parse и compile template;
- text и attribute interpolation;
- conditional и repeated rendering;
- mapping event attributes;
- dynamic attributes и styles;
- pseudo-style directives, например `style-hover`;
- создание React elements;
- mount ReactDOM root;
- placeholders для unresolved values;
- import/component hooks runtime.

React берётся из:

```text
window.React
```

ReactDOM берётся из:

```text
window.ReactDOM
```

Render pipeline:

```text
template
-> parse
-> compile expressions and attributes
-> build render functions
-> create React elements
-> mount into #dc-root
```

## 📄 Синтаксис template

### Interpolation

```html
<span>{{user.name}}</span>
<img src="{{user.avatarUrl}}">
```

Full attribute interpolation сохраняет исходный value type. Mixed interpolation формирует string.

### Conditional rendering

```html
<sc-if value="{{isAuthenticated}}">
  <main>...</main>
</sc-if>
```

False value не рендерит subtree.

### Repeated rendering

```html
<sc-for list="{{posts}}" as="post">
  <article>{{post.text}}</article>
</sc-for>
```

Runtime добавляет `$index` в repeated scope.

### Event handlers

```html
<button onclick="{{submitPost}}">Publish</button>
<input onchange="{{onDraftChange}}">
```

HTML event attributes преобразуются в React event names.

### Dynamic attributes и styles

```html
<button
  disabled="{{pending}}"
  style="background:{{buttonBackground}}"
  style-hover="filter:brightness(1.05)">
```

Values разрешаются из component scope.

### Head content

`<helmet>` преобразуется в runtime head-management directive.

## 📁 Ответственность файлов

| File | Назначение |
|---|---|
| `index.html` | declarative SPA template и script loading |
| `js/runtime.js` | generated bundle `dc-runtime` |
| `js/app.js` | main application state и orchestration screens |
| `js/auth-api.js` | same-origin HTTP client и `APIError` |
| `js/resources.js` | resource/bootstrap integration |
| `js/user-model.js` | user normalization, privacy cleanup, relationship state, request gates |
| `js/post-model.js` | post normalization и state helpers |
| `js/comment-model.js` | comment normalization и pagination merge |
| `js/group-event-model.js` | event normalization и RSVP state |
| `js/notification-model.js` | notification normalization и revision merges |
| `js/chat-model.js` | chat keys, history merge, optimistic/realtime reconciliation |
| `js/avatar-url.js` | classification controlled avatar URLs |
| `css/styles.css` | bundled font declarations |
| `css/styles-2.css` | theme tokens, global styles, animations |
| `Caddyfile` | static delivery, backend proxy, SPA boundary |
| `Dockerfile` | non-root static Caddy image |

Production identities, posts, comments, groups, events, notifications и chats загружаются из backend. `USERS.me` остаётся только current-user holder.

## 🗃️ Application state

`app.js` группирует state по features.

Основные зоны:

```text
theme and current screen
authentication and registration
current user and directory
feed and post composer
per-post comments
profile and profile lists
groups and membership transitions
group posts
group events and RSVP
notifications and unread revision
chats, history, typing, presence, unread revision
```

Feature-specific constructors задают единый reset:

```text
emptyRegistrationForm
emptyProfileEditor
emptyCommentState
emptyGroupPostState
emptyGroupEventState
emptyNotificationState
emptyChatState
```

Comments хранятся per post:

```text
commentsByPostID[postID]
```

Chat histories, unread values и read markers хранятся по canonical chat key.

Backend IDs authoritative. Initials, colors, labels, avatar flags и display time вычисляются локально.

## 🌐 HTTP client

`js/auth-api.js` экспортирует:

```text
APIError
createAuthAPI(fetchImpl)
```

В browser создаётся:

```text
window.AuthAPI
```

Все requests используют:

```text
credentials: same-origin
Accept: application/json
```

Каждый method проверяет exact expected HTTP status.

`APIError` содержит:

```text
message
status
details
cause
```

Network failure имеет status `0`.

Client покрывает:

- registration, login, logout, session restore;
- profile update и avatar replacement;
- users, relationships, followers, follow requests;
- feed, profile posts, group posts, comments с media;
- groups, invitations, join requests;
- events и RSVP;
- notifications;
- chat lists, histories, read markers.

Multipart requests передают `FormData` напрямую и не задают boundary вручную.

## 🔄 Нормализация и authoritative state

Backend objects нормализуются до записи в UI state.

Нормализация включает:

- snake_case в стабильные frontend properties;
- positive backend ID validation;
- canonical store keys;
- безопасную работу с null;
- normalization relationship state;
- classification media/avatar URLs;
- formatting timestamps только для display;
- pagination merge и deduplication.

`UserModel.normalizeUser` удаляет protected data при потере profile access:

```text
email
date of birth
gender
about
statistics
private custom avatar
```

Mutation response считается authoritative. Frontend применяет returned backend state, а не реконструирует security-sensitive result по user intent.

Примеры:

- follow result определяет `accepted` или `pending`;
- group result определяет membership;
- event response возвращает current RSVP counts;
- notification action возвращает resolution, unread и source;
- chat read возвращает authoritative marker и revision.

## 🛡️ Request generations

Async reads используют gates из:

```text
UserModel.createRequestGate()
```

Gate:

```text
begin()
isCurrent(generation)
current()
```

Response меняет state только если generation всё ещё current.

Независимые gates используются для:

- auth bootstrap;
- feed;
- user directory;
- profile, profile posts и lists;
- group directory;
- group detail и membership lists;
- group posts;
- group events и per-event RSVP;
- notifications и notification actions;
- chats и active history;
- per-chat access/read state;
- per-post comment access/load.

Access revoke также хранится явно для group/chat state.

Это не позволяет:

- old profile read вернуть private data после unfollow;
- stale group read вернуть member-only content после leave;
- stale comment create вернуть content после parent access loss;
- stale notification action перезаписать новый lifecycle;
- old chat-list response вернуть удалённый group chat;
- pre-logout request заполнить state следующего user.

Только новый authoritative rejoin response может снять group revoke.

## 🔑 Authentication lifecycle

Startup state:

```text
authStatus: checking
```

Frontend вызывает:

```text
GET /api/auth/me
```

Варианты:

- valid session: current user и bootstrap application data;
- `401`: login/registration;
- network/server error: retryable startup error.

Registration использует `FormData`, login использует JSON.

После authentication:

- returned user становится `USERS.me`;
- обновляется normalized user store;
- начинается загрузка feature data;
- запускается WebSocket.

Logout:

- вызывает backend endpoint;
- invalidates request generations;
- закрывает realtime;
- очищает user-derived collections;
- очищает drafts и selected files;
- revoke preview object URLs;
- сбрасывает current-user holder;
- возвращает auth screen.

Failed logout не маскируется как удалённая server session.

## 📝 Posts, comments и media

### Post composers

Personal post state:

```text
text
optional File
privacy
selected follower IDs
pending/error
```

Group post composer имеет отдельный state и gates.

Posts и comments создаются через `FormData`.

### Per-post comments

Для каждого post независимо хранятся:

```text
comments
cursor
loading
load pending
load error
draft
media File
media filename
preview URL
create pending
create error
access gate
load gate
```

File input ID уникален:

```text
comment-media-{postID}
```

Create behavior:

- send требует non-empty text;
- media optional;
- file controls disabled при pending create;
- validation, network и server errors сохраняют text, `File`, preview;
- `403` или `404` считаются access revoke и очищают attachment;
- success применяет authoritative comment и очищает composer;
- late result игнорируется при stale access generation.

### Object URL lifecycle

При замене file:

```text
old URL -> revoke
new File -> URL.createObjectURL
```

Preview URL revoke при:

- replacement;
- ручном remove;
- successful create;
- parent access revoke;
- purge comment state;
- logout;
- component unmount.

Controls disabled во время pending create, поэтому file, выбранный позже, не может быть удалён success handler старого request.

## 👥 Groups и events

Group state разделён на:

```text
directory
invitation inbox
selected group detail
members
join requests
sent invitations
group posts
group events
RSVP pending/error per event
membership mutation pending/error per group
```

UI permissions выводятся из authoritative `viewer_status`.

Owner-only controls:

- join-request management;
- sent invitation management.

Owner/member controls:

- invite;
- group posts/comments;
- event creation и RSVP;
- group chat.

Leave или realtime `chat:remove`:

- invalidates group generations;
- скрывает member-only controls;
- purges posts, comments, events, drafts и previews;
- удаляет group chat;
- блокирует stale responses.

## 🔔 Notifications

Notification state:

```text
items and cursor
loading/pagination flags
unread count
revision
per-item read state
per-item action state
mark-all state
```

HTTP и persisted revisions authoritative. Realtime events используются как upsert/refresh signals.

Frontend:

- merge items по notification ID;
- применяет только non-stale revisions;
- refresh после actions и reconnect;
- race-gates source transitions отдельно;
- поддерживает mark-one, mark-all, accept и decline;
- обновляет navigation badge из persisted unread.

## 💬 Chat и realtime

Chat state keyed по canonical direct/group key.

```text
summaries
active chat
history per key
presence
typing
optimistic messages
per-chat unread
total unread and revision
read markers
WebSocket status
reconnect attempt
```

После authenticated bootstrap frontend открывает same-origin `/ws`.

Handled events:

```text
presence:init
presence:update
presence:remove
typing:update
chat:message
chat:unread
chat:remove
chat:error
notification:upsert
notifications:read-all
```

Reconnect reloads chat list и active authoritative history.

### Optimistic send

Новое message получает client UUID и показывается сразу.

States:

```text
pending
sent
failed
```

Retry использует тот же UUID. History и realtime responses дедуплицируются по client/server identity. Unacknowledged optimistic message после timer становится retryable failed.

### Read behavior

Chat mark read только когда:

- Messages active screen;
- conversation active;
- authoritative history отображена;
- browser tab visible.

Не mark read:

- bootstrap;
- reconnect preload;
- background list refresh;
- inactive history;
- older-history pagination.

Read markers и revisions двигаются только вперёд.

### Typing и presence

Typing events:

```text
typing:start
typing:heartbeat
typing:stop
```

Heartbeat и expiry timers не дают typing indicator зависнуть. Visibility и access changes удаляют stale presence/typing.

## 🧹 Очистка ресурсов

`componentWillUnmount`:

- remove `visibilitychange`;
- revoke remaining comment previews;
- stop realtime.

Logout и access revoke дополнительно очищают:

- pending и typing timers;
- WebSocket reconnect timer;
- active socket;
- optimistic messages и chat drafts;
- per-post files/previews;
- group content/drafts;
- notification и user state;
- stale request generations.

Цель - не допустить переноса state одного пользователя в следующую session.

## 📦 Production delivery

Production image содержит только:

```text
index.html
css/
js/
assets/
Caddyfile
```

Node, tests и frontend build tooling не входят.

Caddy proxies exact и wildcard paths:

```text
/api
/api/*
/ws
/ws/*
/static/avatars
/static/avatars/*
```

Legacy paths получают `404` до SPA fallback:

```text
/uploads
/uploads/*
```

Остальные paths обслуживаются статически с `index.html` fallback.

Caddy работает numeric non-root user. Writable config/data используют `/tmp`, root filesystem read-only.

Build и launch commands находятся в [корневом README](../README_RU.md).

## 🧪 Test suite

```bash
npm test
```

Файлы:

```text
js/app-race.test.js
js/auth-api.test.js
js/avatar-url.test.js
js/chat-model.test.js
js/comment-model.test.js
js/group-event-model.test.js
js/notification-model.test.js
js/post-model.test.js
js/user-model.test.js
```

Покрываются:

- exact HTTP methods/statuses;
- API и network errors;
- URL classification;
- response normalization;
- privacy field removal;
- request generations;
- pagination merge/dedup;
- selected audience pruning;
- comment create и preview lifecycle;
- events и RSVP;
- notification revisions/source races;
- optimistic chat send/retry;
- reconnect/history merge;
- unread/read-marker monotonicity;
- logout/access-revoke stale-response protection.

Accepted result:

```text
121/121 tests passed
```

## 📁 Структура

```text
frontend/
├── assets/
│   ├── fonts/
│   └── ...
├── css/
│   ├── styles.css
│   └── styles-2.css
├── js/
│   ├── app.js
│   ├── app-race.test.js
│   ├── auth-api.js
│   ├── auth-api.test.js
│   ├── avatar-url.js
│   ├── chat-model.js
│   ├── comment-model.js
│   ├── group-event-model.js
│   ├── notification-model.js
│   ├── post-model.js
│   ├── resources.js
│   ├── runtime.js
│   ├── user-model.js
│   └── vendor/
├── Caddyfile
├── Dockerfile
├── index.html
├── package.json
├── README.md
└── README_RU.md
```
