# Кроссплатформенный desktop-клиент

Desktop-клиент на Electron для проекта `social-network`. Реализация находится в `desktop/` и использует существующие frontend, HTTP API, sessions, GitHub authentication flow и WebSocket endpoint.

· [English version](README.md)

Готовые бинарные файлы в репозитории не хранятся. Результат сборки создаётся в `desktop/release/`, этот каталог исключён через `.gitignore`.

## Возможности

- сохранение авторизации между перезапусками приложения до истечения backend session или ручного logout;
- вход по email/password через существующий backend;
- вход через GitHub с использованием существующего server-side OAuth flow, если GitHub OAuth настроен;
- realtime direct/group chat и presence через существующий `/ws`;
- системные desktop-уведомления о входящих сообщениях;
- emoji из существующего интерфейса чата;
- регистрация открывает обычный сайт social-network в браузере;
- предупреждение об offline-состоянии, просмотр закэшированной истории и блокировка отправки без соединения;
- интерактивный поиск сообщений без кнопки, результаты меняются сразу при вводе;
- текстовые операторы include, exclude и fuzzy;
- числовые операторы равенства и сравнения;
- сборка под Windows, Linux AppImage и macOS.

## Требования

- backend/frontend social-network должны быть запущены и доступны;
- Node.js 22 или новее и npm для запуска и сборки desktop-клиента.

Запуск существующего приложения из корня репозитория:

```bash
docker compose up --build
```

По умолчанию desktop-клиент подключается к:

```text
http://127.0.0.1:8080
```

## Запуск без упаковки

Из корня репозитория:

```bash
cd desktop
npm install
npm run dev
```

`npm run dev` сначала собирает актуальный frontend, затем запускает Electron.

## Сборка установщика/пакета

Один раз установить зависимости:

```bash
cd desktop
npm install
```

Сборка для текущей платформы:

```bash
npm run dist
```

Или выбрать платформу явно:

```bash
npm run dist:win
npm run dist:linux
npm run dist:mac
```

Результат появляется в:

```text
desktop/release/
```

Ожидаемые имена пакетов:

- Windows: `Loop-<version>-Setup.exe`
- Linux: `Loop-<version>.AppImage`
- macOS: `Loop-<version>.dmg`

Пакет следует собирать на соответствующей операционной системе. Готовые пакеты намеренно не коммитятся в Git, поэтому ограничения хостинга на размер файлов на них не влияют.

## Подключение к другому серверу

Перед запуском задайте `SOCIAL_NETWORK_URL`. `SOCIAL_NETWORK_WEB_URL` отдельно задаёт сайт, который открывается для регистрации.

PowerShell:

```powershell
$env:SOCIAL_NETWORK_URL = "http://192.168.1.20:8080"
$env:SOCIAL_NETWORK_WEB_URL = "http://192.168.1.20:8080"
npm run dev
```

Bash:

```bash
SOCIAL_NETWORK_URL=http://192.168.1.20:8080 \
SOCIAL_NETWORK_WEB_URL=http://192.168.1.20:8080 \
npm run dev
```

Если переменные не заданы, используются значения `http://127.0.0.1:8080`.

## Вход через GitHub

Desktop-клиент использует GitHub flow, уже реализованный в веб-приложении. Flow открывается в отдельном Electron-окне с тем же persistent partition. После создания backend session desktop-слой переносит полученные cookies на loopback-origin основного окна и перезагружает приложение.

Client ID и client secret не встраиваются в desktop-приложение и не коммитятся в репозиторий. GitHub-вход доступен, когда backend настроен через:

```text
SOCIAL_NETWORK_GITHUB_CLIENT_ID
SOCIAL_NETWORK_GITHUB_CLIENT_SECRET
SOCIAL_NETWORK_GITHUB_REDIRECT_URL
```

Callback URL должен указывать на:

```text
/api/auth/oauth/github/callback
```

Если эти переменные пусты, backend не публикует GitHub provider и кнопка GitHub не отображается.

## Поиск сообщений

Поиск пересчитывается сразу после каждого изменения строки ввода.

Обычные слова работают как обязательный substring-фильтр. Дополнительно поддерживаются:

```text
+слово              включить
include:слово       включить
-слово              исключить
exclude:слово       исключить
~слово              нечёткий поиск
fuzzy:слово         нечёткий поиск
```

Для фраз с пробелами можно использовать кавычки:

```text
"release candidate"
exclude:"not ready"
```

Числа, содержащиеся в тексте сообщений, можно сравнивать операторами:

```text
=10
!=10
>10
<10
>=10
<=10
```

Операторы можно комбинировать, например:

```text
+invoice -rejected >100 <200
```

Такой запрос оставляет сообщения с `invoice`, исключает `rejected` и проверяет числовые значения по заданным границам.

## Тесты

Из `desktop/`:

```bash
npm test
```

Тесты покрывают local proxy, offline cache fallback, session cache seeding, переписывание cookies, WebSocket proxy headers, передачу GitHub session между origins, fuzzy/text operators и числовые операторы поиска.

Frontend можно проверить отдельно:

```bash
cd ../frontend
npm test
npm run build
```

## Ручная проверка

Используйте двух пользователей: например, один работает в обычном браузере, второй — в desktop-клиенте.

1. Войдите в desktop-клиент.
2. Закройте и снова откройте его — session должна сохраниться.
3. Отправьте сообщение browser → desktop и desktop → browser; оба направления должны обновляться realtime.
4. Проверьте изменение online/offline presence без перезагрузки.
5. Получите сообщение, когда окно Loop не в фокусе, и проверьте системное уведомление.
6. Отключите сеть: должно появиться предупреждение, история должна остаться доступной, отправка должна блокироваться.
7. Верните сеть и проверьте восстановление realtime-работы.
8. Откройте Messages и вводите текст в поле поиска — результаты должны меняться сразу.
9. Проверьте include, exclude, fuzzy и числовые операторы в том же поле.
10. Если GitHub OAuth настроен, используйте `Continue with GitHub` и проверьте, что desktop-session сохраняется после перезапуска Loop.
11. Выполните logout — он должен оставаться доступным из интерфейса приложения.

## Детали реализации

Electron отдаёт собранный frontend через loopback-only HTTP server и проксирует `/api`, `/static/avatars` и `/ws` на настроенный origin social-network. Благодаря этому сохраняется существующая same-origin модель frontend/API без отдельной копии приложения.

Electron использует persistent partition `persist:loop`. Desktop-функции доступны renderer через sandboxed preload bridge с включённым `contextIsolation` и отключённым Node integration.

Пакеты по умолчанию не подписываются. Поэтому операционная система может показывать стандартное предупреждение для локально собранного неподписанного приложения.

## Авторы

Nazar Yestayev (@nyestaye)  
Nurislam Danbaev (@ndanbaev)  
Amir Zhakyshev (@azhakysh)  
Magzhan Tastan (@mtastan)  
Kuanysh Karimov (@kukarimov)  
Нұрайдар Мәмбеталы (@nmambetal)
