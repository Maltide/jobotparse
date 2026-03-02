## Проект на текущий момент
- Реализовано: поиск вакансий (SuperJob OAuth) и UI `/vacancies`
- UI чат с ИИ-моделью, создание резюме `/adapt`

TODO:
- Не реализовано в роутинге: `POST /adapt/finalize` (в UI кнопка есть, но в Go‑роутах сейчас опубликован `/final`).

## Описание
- `app` — Go сервис 
- `postgres` — база данных
- `nginx` — обратный прокси сервер (требует свободные порты 80/443)

## Требования
- Docker Engine
- Docker Compose v2 (плагин `docker compose`)

## Старт с ПК
Создать файл `.env` (он используется как `env_file` для контейнера).
   `PG_HOST=postgres`, `PG_PORT=5432`, `PG_SSLMODE=disable`, `BASE_URL=...`, `ADMIN_USER/ADMIN_PASS`, `CLIENT_ID/CLIENT_SECRET`(берется на сайте API).

   `BASE_URL` участвует в OAuth Superjob API и на него происходит редирект (например `${BASE_URL}/callback`).

Запуск:
- `docker compose up -d --build`

Токены SuperJob сохраняются в `./data/tokens.json` (папка `data/` игнорируется git).

## Доступ из сети / домен
Если OAuth привязан к домену, то запросы на `https://<домен>/callback` должны приходить на этот ПК.

## Nginx из compose + certbot 
Тут Nginx выступает как **reverse-proxy (обратный прокси)** — принимает запросы на 80/443 и прокидывает внутрь на приложение (у нас на 8080).

**certbot** кладёт сертификаты в `/.well-known/acme-challenge/`

Запуск прокси:
- `docker compose up -d --build nginx`

Получение сертификата (один раз):
- `docker compose run --rm certbot certonly --webroot -w /var/www/certbot -d <твой-домен> --agree-tos --email <твоя-почта> --no-eff-email`

После получения сертификата перезапустить nginx:
- `docker compose restart nginx`

Сертификаты и состояния certbot хранятся в папке `certbot/` (она в `.gitignore`).

- `http://localhost:8080/auth` - обновляет токен. Переходит на ui API Superjob, который редиректит на указанный тобой URL
- `http://localhost:8080/vacancies` - поиск вакансий по фильтрам - при заполнении отправляет POST-запрос с фильтрами к API Superjob вместе с ключом доступа к API, возвращает вакансии, которые будут отображаться в UI

## ИИ
Текущее взаимодействие UI ↔ Go‑бэкенд ↔ ИИ‑модель (Ollama API) и дальнейшая конвертация JSON → HTML → PDF.

- UI форма + чат: [static/adapt.html](static/adapt.html)
- Инструкция модели для адаптации: [static/aireq.txt](static/aireq.txt)
- Сборка текста резюме для промпта: [pkg/helpers/resume.go](pkg/helpers/resume.go)
- HTTP‑роуты: [pkg/server/server.go](pkg/server/server.go)
- Эндпоинты: [pkg/handlers/handler.go](pkg/handlers/handler.go)
- Шаблон для получения JSON-структуры с инструкциями [static/jsonreq.txt](static/jsonreq.txt)

### Эндпоинты
- `GET /adapt` — отдаёт страницу формы/чата.
- `POST /adapt` — старт сессии: получает JSON резюме, подтягивает вакансию, собирает prompt и делает 1-й запрос к модели.
- `POST /adapt/iterate` — переводит с /adapt на чат, в котором можно прописать доп инструкции или редактировать заполненное резюме(в этом случае диалоговая сессия начнется заново). TODO: допилить реализацию "Конвертировать в PDF"



