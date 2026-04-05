FROM nginx:alpine

# Özel Nginx konfigürasyonunu kopyala
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Web dosyalarını Nginx'in sunacağı dizine al
COPY src/ /usr/share/nginx/html/

EXPOSE 80
