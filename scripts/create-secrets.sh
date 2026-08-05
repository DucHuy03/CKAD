#!/usr/bin/env bash
# Creates runtime Kubernetes Secrets from local environment variables.
# No credential value is stored in this repository. Export values first, for
# example: export DB_PASSWORD='change-me' JWT_SECRET='change-me'.

set -euo pipefail
NAMESPACE="${NAMESPACE:-movie-booking}"
: "${DB_PASSWORD:?Set DB_PASSWORD before creating database secrets}"
: "${JWT_SECRET:?Set JWT_SECRET before creating gateway secret}"
: "${RABBITMQ_PASSWORD:?Set RABBITMQ_PASSWORD before creating RabbitMQ secret}"
: "${RABBITMQ_USERNAME:?Set RABBITMQ_USERNAME before creating RabbitMQ secret}"

for service in movie booking payment notification; do
  kubectl create secret generic "postgres-$service-secret" -n "$NAMESPACE" \
    --from-literal=POSTGRES_USER=postgres --from-literal=POSTGRES_PASSWORD="$DB_PASSWORD" \
    --dry-run=client -o yaml | kubectl apply -f -
  kubectl create secret generic "$service-service-secret" -n "$NAMESPACE" \
    --from-literal=DB_USER=postgres --from-literal=DB_PASSWORD="$DB_PASSWORD" \
    --dry-run=client -o yaml | kubectl apply -f -
done

# Payment and notification own the broker connection string because it embeds
# credentials. The URL is assembled locally and is never committed.
for service in payment notification; do
  kubectl create secret generic "$service-service-secret" -n "$NAMESPACE" \
    --from-literal=DB_USER=postgres --from-literal=DB_PASSWORD="$DB_PASSWORD" \
    --from-literal=RABBITMQ_URL="amqp://${RABBITMQ_USERNAME}:${RABBITMQ_PASSWORD}@rabbitmq:5672/" \
    --dry-run=client -o yaml | kubectl apply -f -
done

kubectl create secret generic rabbitmq-secret -n "$NAMESPACE" \
  --from-literal=RABBITMQ_DEFAULT_USER="$RABBITMQ_USERNAME" --from-literal=RABBITMQ_DEFAULT_PASS="$RABBITMQ_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret generic api-gateway-secret -n "$NAMESPACE" --from-literal=JWT_SECRET="$JWT_SECRET" \
  --dry-run=client -o yaml | kubectl apply -f -
