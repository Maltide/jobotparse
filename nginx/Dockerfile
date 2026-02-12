# Используем официальный образ nginx
FROM nginx:stable

# Генерируем конфиг при старте контейнера:
# - если сертификата ещё нет, запускаем только HTTP (нужно для ACME HTTP-01)
# - когда certbot положит сертификат, контейнер на рестарте включит HTTPS
COPY templates/ /etc/nginx/templates/
COPY docker-entrypoint.d/ /docker-entrypoint.d/

# (Опционально) Копируем статику, если нужно
# COPY ../static/ /usr/share/nginx/html/

# Сертификаты будут пробрасываться через volume, поэтому отдельный COPY не нужен