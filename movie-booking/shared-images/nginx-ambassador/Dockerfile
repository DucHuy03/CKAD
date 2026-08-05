# Dockerfile - nginx-ambassador.
#
# Dung "nginx-unprivileged" thay vi "nginx" thong thuong: image nay da cau
# hinh san de chay KHONG CAN quyen root (listen port >1024, khong bind port
# 80 mac dinh vi port <1024 tren Linux can quyen root) - chuan bi san cho
# Lab 3.2 (Security Context: runAsNonRoot), khong phai sua lai sau.
#
# KHONG can viet entrypoint rieng: image nginx (ca ban unprivileged) co san
# co che "envsubst on templates" - moi file *.template trong
# /etc/nginx/templates/ se duoc entrypoint mac dinh chay envsubst() thay the
# cac bien moi truong (${UPSTREAM_HOST}, ${UPSTREAM_PORT}, ${LISTEN_PORT})
# ngay khi container start, xuat ra /etc/nginx/conf.d/default.conf - moi
# service chi can set 3 bien moi truong nay khac nhau khi chay container,
# KHONG can build 3 image nginx khac nhau.
FROM nginxinc/nginx-unprivileged:1.27-alpine

COPY default.conf.template /etc/nginx/templates/default.conf.template

# Port mac dinh - co the doi qua bien LISTEN_PORT (dong bo voi template),
# nhung EXPOSE trong Dockerfile chi la "tai lieu", khong ep buoc runtime.
EXPOSE 8080