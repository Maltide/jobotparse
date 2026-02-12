# jobotparse — локальный запуск на ПК

## Что запускается
- `app` — Go сервис на `http://localhost:8080`
- `postgres` — база в Docker volume (наружу порт **не** публикуется)
- `nginx` — опционально (profile `prod`), требует свободные порты 80/443 и сертификаты

## Требования
- Docker Engine
- Docker Compose v2 (плагин `docker compose`)

Примечание: `docker-compose` v1 (Python) может падать с Docker Engine 28+ (ошибка вроде `KeyError: 'ContainerConfig'`).

### Установка Compose v2 без sudo (рекомендуется)
Если нет `sudo`, можно поставить плагин в домашнюю папку:
- `mkdir -p ~/.docker/cli-plugins`
- `curl -fL https://github.com/docker/compose/releases/download/v2.33.0/docker-compose-linux-x86_64 -o ~/.docker/cli-plugins/docker-compose`
- `chmod +x ~/.docker/cli-plugins/docker-compose`
- `docker compose version`

## Быстрый старт (ПК)
1) Создай файл `.env` (он используется как `env_file` для контейнера).
   Минимально важное: `PG_HOST=postgres`, `PG_PORT=5432`, `PG_SSLMODE=disable`, `BASE_URL=...`, `ADMIN_USER/ADMIN_PASS`, `CLIENT_ID/CLIENT_SECRET`.

   Важно: `BASE_URL` участвует в OAuth redirect (например `${BASE_URL}/callback`).
   Если OAuth у тебя привязан к домену — оставляй `BASE_URL=https://<твой-домен>`.

2) Запусти:
- `docker compose up -d --build`

3) Токены SuperJob сохраняются в `./data/tokens.json` (папка `data/` игнорируется git).

## Доступ из сети / домен
Если OAuth привязан к домену, то запросы на `https://<домен>/callback` должны приходить на этот ПК.

Есть два варианта:

### Вариант A: Nginx из compose + certbot (profile `prod`) на 80/443
Тут Nginx выступает как **reverse-proxy (обратный прокси)** — принимает запросы на 80/443 и прокидывает внутрь на приложение (у нас на 8080).

Сертификаты делает **certbot** — утилита для Let's Encrypt. Она кладёт challenge-файлы в `/.well-known/acme-challenge/`, а Nginx отдаёт их на 80.

Важно:
- Порты `80/443` на хосте должны быть свободны (Apache остановлен/отключён или перенесён на другие порты).
- Если ПК за роутером (**NAT** — когда у ПК адрес вида `192.168.x.x`, а наружу выходит один «общий» публичный IP), нужно сделать **port forwarding / проброс портов** на роутере: TCP `80` и TCP `443` → на IP твоего ПК в локалке.
- Если провайдер использует **CGNAT** (когда у роутера нет «настоящего» публичного IPv4), прямой вход с интернета может быть невозможен — тогда нужен туннель (например Cloudflare Tunnel) или выделенный публичный IP у провайдера.

Запуск прокси:
- `docker compose --profile prod up -d --build nginx`

Получение сертификата (один раз):
- `docker compose --profile prod run --rm certbot certonly --webroot -w /var/www/certbot -d <твой-домен> --agree-tos --email <твоя-почта> --no-eff-email`

После получения сертификата перезапусти nginx:
- `docker compose --profile prod restart nginx`

Сертификаты и состояния certbot хранятся в папке `certbot/` (она в `.gitignore`).

### Вариант B: Apache как reverse-proxy
Если нужно оставить Apache, его можно настроить как reverse-proxy на `http://127.0.0.1:8080` и TLS держать на Apache. Это вариант для случаев, когда у тебя уже есть рабочие конфиги Apache/сертификаты и ты не хочешь заводить Nginx.

3) Открой:
- `http://localhost:8080/auth`
- `http://localhost:8080/vacancies`
- `http://localhost:8080/adapt`

## Ollama (если работает на этом же ПК)
Контейнер обращается к Ollama по `http://host.docker.internal:11434`.
На Linux это включено через `extra_hosts: host.docker.internal:host-gateway`.

Можно переопределить адрес и модель через переменные:
- `OLLAMA_BASE_URL`
- `OLLAMA_MODEL`

## Nginx (опционально)
По умолчанию `nginx` **не запускается**, чтобы не конфликтовать с Apache/занятыми портами 80/443.

Если хочешь включить:
- `docker compose --profile prod up -d --build`

Важно: текущий конфиг nginx ожидает сертификаты в `certbot/conf` (они монтируются внутрь контейнера как `/etc/letsencrypt`) и доменное имя в `nginx/nginx.conf`.
