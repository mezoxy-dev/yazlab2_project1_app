FROM nginx:alpine 
# Dashboard gibi bir uygulama genellikle statik dosyalardan oluşur, bu yüzden Nginx ideal bir seçimdir.
COPY nginx.conf /etc/nginx/conf.d/default.conf 
COPY index.html /usr/share/nginx/html/index.html
COPY app.js /usr/share/nginx/html/app.js
COPY style.css /usr/share/nginx/html/style.css
EXPOSE 80
 