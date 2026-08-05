# Dockerfile - log-shipper sidecar.
#
# Chi can alpine (co san busybox voi lenh tail ho tro -F) la du, khong can
# Go/bat ky runtime nao khac - day la mot sidecar CO CHU DICH DON GIAN, cang
# it dependency cang giam bem mat tan cong va kich thuoc image (~7MB).

FROM alpine:3.19

COPY entrypoint.sh /entrypoint.sh
# UID/GID so cu the de Kubernetes xac minh duoc image non-root khi Pod
# bat runAsNonRoot; username "app" mot minh khong du de kubelet xac minh.
RUN chmod +x /entrypoint.sh \
    && addgroup -g 10001 app && adduser -D -u 10001 -G app app

USER 10001

ENTRYPOINT ["/entrypoint.sh"]
