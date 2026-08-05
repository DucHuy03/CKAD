#!/usr/bin/env bash
# Dong bo user RabbitMQ trong PVC voi Secret Kubernetes hien tai.
# RabbitMQ chi doc RABBITMQ_DEFAULT_* trong LAN KHOI TAO data directory dau
# tien; neu Secret doi sau do, password trong PVC cu se khong tu cap nhat.
# Script nay dung rabbitmqctl NOI BO Pod, khong in gia tri bi mat ra terminal.

set -euo pipefail

NAMESPACE="${NAMESPACE:-movie-booking}"
SECRET_NAME="${RABBITMQ_SECRET_NAME:-rabbitmq-secret}"
DEPLOYMENT="${RABBITMQ_DEPLOYMENT:-rabbitmq}"

decode_secret_key() {
  kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" \
    -o "jsonpath={.data.$1}" | base64 --decode
}

USERNAME="$(decode_secret_key RABBITMQ_DEFAULT_USER)"
PASSWORD="$(decode_secret_key RABBITMQ_DEFAULT_PASS)"

# list_users khong can mat khau khi chay ben trong chinh RabbitMQ container.
if kubectl exec -n "$NAMESPACE" "deployment/$DEPLOYMENT" -- \
  rabbitmqctl list_users | awk -v user="$USERNAME" '$1 == user { found = 1 } END { exit !found }'; then
  kubectl exec -n "$NAMESPACE" "deployment/$DEPLOYMENT" -- \
    rabbitmqctl change_password "$USERNAME" "$PASSWORD"
else
  kubectl exec -n "$NAMESPACE" "deployment/$DEPLOYMENT" -- \
    rabbitmqctl add_user "$USERNAME" "$PASSWORD"
fi

# Service can publish/consume tren vhost mac dinh, khong can tag administrator.
kubectl exec -n "$NAMESPACE" "deployment/$DEPLOYMENT" -- \
  rabbitmqctl set_permissions -p / "$USERNAME" '.*' '.*' '.*'

echo "RabbitMQ application credentials reconciled."
