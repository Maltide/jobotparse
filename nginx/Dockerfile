# Используем официальный образ nginx
FROM nginx:stable

# Копируем свой конфиг nginx внутрь контейнера
COPY nginx.conf /etc/nginx/nginx.conf

# (Опционально) Копируем статику, если нужно
# COPY ../static/ /usr/share/nginx/html/

# Сертификаты будут пробрасываться через volume, поэтому отдельный COPY не нужен