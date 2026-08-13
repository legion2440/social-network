# Кроссплатформенный desktop-клиент

Desktop-клиент на Electron для проекта `social-network`. Реализация находится в `desktop/` и использует существующие frontend, HTTP API, sessions, GitHub authentication flow и WebSocket endpoint.

· [English version](README.md)

Готовые бинарные файлы в репозитории не хранятся. Результат сборки создаётся в `desktop/release/`, этот каталог исключён через `.gitignore`.

## Возможности

- сохранение авторизации до истечения backend session или ручного logout;
- вход по email/password через существующий backend;
- вход через GitHub через существующий server-side OAuth flow, если он настроен;
- realtime direct/group chat и presence;
- системные уведомления о входящих сообщениях;
- emoji из существующего интерфейса чата;
- регистрация открывает сайт сразу в режиме создания аккаунта;
- предупреждение об offline-состоянии, просмотр закэшированной истории и блокировка отправки без соединения;
- автоматическое восстановление после возвращения настроенного сервера;
- интерактивный поиск сообщений с автоматической загрузкой более старой истории;
- операторы include, exclude, fuzzy, равенства и числового сравнения;
- сборка установщика Windows, Linux AppImage и macOS DMG.

## Требования

- Node.js 22 или новее;
- npm;
- запущенные backend/frontend `social-network`, доступные с компьютера, на котором работает Loop.

Если web-приложение запускается на той же машине, из корня репозитория:

```bash
docker compose up --build
```

## Подключение к серверу

При первом запуске, если сервер ещё не сохранён и `SOCIAL_NETWORK_URL` не задан, Loop показывает отдельное окно **Connect to your server**. Поле заранее заполнено адресом:

```text
http://127.0.0.1:8080
```

Укажите адрес, доступный с текущего компьютера или виртуальной машины, и нажмите **Connect**. Перед сохранением Loop проверяет `/api/health`. Настройка хранится в каталоге user data Electron.

В обычном интерфейсе адрес сервера не показывается. Позже его можно изменить через **Settings → Server…**. В Windows и Linux системное меню может быть скрыто до нажатия `Alt`. В macOS используется **Loop → Server Settings…**. После смены сервера Loop перезапускается.

Если ранее настроенный сервер временно недоступен, приложение запускается в offline-режиме, а не возвращает пользователя на экран настройки. Loop периодически проверяет сервер и после восстановления соединения автоматически обновляет приложение, поэтому старые сообщения об ошибке исчезают без ручного F5.

`SOCIAL_NETWORK_URL` остаётся явным override для разработки или управляемого запуска. Если переменная задана, она имеет приоритет над сохранённой настройкой, а поле Server Settings становится read-only. `SOCIAL_NETWORK_WEB_URL` может отдельно задавать сайт, открываемый для регистрации; если переменная не задана, используется тот же сервер.

Пример PowerShell:

```powershell
$env:SOCIAL_NETWORK_URL = "http://192.168.1.20:8080"
$env:SOCIAL_NETWORK_WEB_URL = "http://192.168.1.20:8080"
npm run dev
```

Пример Bash:

```bash
SOCIAL_NETWORK_URL=http://192.168.1.20:8080 \
SOCIAL_NETWORK_WEB_URL=http://192.168.1.20:8080 \
npm run dev
```

## Запуск без упаковки

Из корня репозитория:

```bash
cd desktop
npm ci
npm run dev
```

`npm run dev` сначала собирает актуальный frontend, затем запускает Electron.

## Сборка установщика или пакета

Установить зафиксированные зависимости:

```bash
cd desktop
npm ci
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

Пакет следует собирать на соответствующей операционной системе. По умолчанию пакеты не подписываются, поэтому ОС может показать стандартное предупреждение для локально собранного приложения.

## Вход через GitHub

Desktop-клиент использует GitHub flow, уже реализованный в web-приложении. Flow открывается в отдельном Electron-окне с persistent desktop session. Навигация этого окна ограничена `github.com` и настроенным origin social-network. После создания backend session Loop переносит полученные session cookies на локальный origin приложения и перезагружает основное окно.

GitHub client ID и client secret не встраиваются в desktop-приложение и не коммитятся в репозиторий. GitHub-вход доступен, когда backend настроен через:

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

## Offline-режим

Когда настроенный сервер недоступен:

- Loop показывает offline-состояние;
- ранее закэшированная история чатов остаётся доступной;
- попытка отправить сообщение перехватывается и показывает то же сообщение об отсутствии соединения;
- realtime sockets закрываются до восстановления соединения.

Локальный HTTP-кэш хранит тела ответов, необходимые для offline-доступа к чатам, но не сохраняет authentication-related response headers. Session cookies остаются в cookie store Electron и не записываются в `offline-cache/http-cache.json`. Закэшированный `/api/auth/me` истекает через 24 часа — тот же срок, что и backend session. Остальные chat cache entries также имеют ограниченный срок хранения. Старые форматы кэша очищаются при загрузке.

## Поиск сообщений

Поиск пересчитывается сразу после каждого изменения строки. Если в активном диалоге есть более старые страницы, Loop автоматически загружает их, пока запрос поиска активен, и обновляет количество совпадений по мере загрузки истории.

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

Числа в тексте сообщений можно сравнивать операторами:

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

## Тесты

Из `desktop/`:

```bash
npm test
```

Desktop-тесты покрывают local proxy, миграцию и срок жизни offline cache, удаление authentication headers из кэша, WebSocket proxy headers, настройки сервера, передачу GitHub session и ограничения OAuth navigation, поисковые операторы и integration hooks frontend.

Frontend можно проверить отдельно:

```bash
cd ../frontend
npm test
npm run build
```

## Ручная проверка

Для end-to-end проверки удобно использовать одну учётную запись в обычном браузере и вторую в Loop. Проверьте сохранение session после перезапуска, realtime-сообщения в обе стороны, presence, системные уведомления, offline-историю и блокировку отправки, автоматическое восстановление после возвращения сервера, поиск по старой истории, GitHub-вход при наличии конфигурации и logout.

## Детали реализации

Electron отдаёт собранный frontend через loopback-only HTTP server и проксирует `/api`, `/static/avatars` и `/ws` на настроенный origin social-network. Благодаря этому сохраняется существующая same-origin модель frontend/API без второй реализации приложения.

Основное окно использует persistent partition `persist:loop`. Desktop-функции доступны renderer через sandboxed preload bridges с включённым `contextIsolation` и отключённым Node integration. Связь desktop adapter с chat template построена через стабильные `data-loop-*` hooks, а не через сгенерированные порядковые CSS-классы.
