#!/usr/bin/env bash
# Tao .env.local chua secret cho may local. File nay da duoc .gitignore bo qua.
# Ten va ngay sinh chi duoc dung lam nhan de de nhan biet; moi secret van co
# phan ngau nhien cryptographic de khong the doan chi tu thong tin ca nhan.
# Usage: bash scripts/generate-local-secrets.sh "Ten Cua Ban" 0311

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="${ENV_FILE:-$PROJECT_ROOT/.env.local}"
DISPLAY_NAME="${1:?Usage: $0 \"Ten Cua Ban\" [ngay_sinh_4_chu_so]}"
BIRTH_CODE="${2:-0311}"

if [[ ! "$BIRTH_CODE" =~ ^[0-9]{4}$ ]]; then
  echo "Ngay sinh phai co 4 chu so, vi du 0311." >&2
  exit 2
fi

if [[ -e "$ENV_FILE" ]]; then
  echo "$ENV_FILE da ton tai; khong ghi de secret hien co." >&2
  echo "Doi ENV_FILE neu muon tao file khac." >&2
  exit 1
fi

command -v openssl >/dev/null || { echo "Thieu openssl de tao random secret." >&2; exit 1; }

# Chuyen ten thanh nhan an toan cho username/password, khong giu ky tu dac biet.
NAME_TAG="$(printf '%s' "$DISPLAY_NAME" | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '_' | sed 's/^_*//; s/_*$//')"
NAME_TAG="${NAME_TAG:-local_user}"
RANDOM_DB="$(openssl rand -hex 16)"
RANDOM_RABBIT="$(openssl rand -hex 16)"
RANDOM_JWT="$(openssl rand -base64 48 | tr -d '\n')"

# umask 077 giup file chi doc/ghi duoc boi tai khoan hien tai tren Unix/WSL.
umask 077
printf '%s\n' \
  "# Generated locally; do not commit this file." \
  "export DB_PASSWORD='${NAME_TAG}_${BIRTH_CODE}_db_${RANDOM_DB}'" \
  "export JWT_SECRET='${RANDOM_JWT}'" \
  "export RABBITMQ_USERNAME='${NAME_TAG}_${BIRTH_CODE}'" \
  "export RABBITMQ_PASSWORD='${NAME_TAG}_${BIRTH_CODE}_mq_${RANDOM_RABBIT}'" \
  > "$ENV_FILE"

echo "Da tao $ENV_FILE (gia tri secret khong duoc in ra)."
echo "Nap secret: source .env.local"
echo "Sau do deploy: bash scripts/deploy-kubernetes.sh"
