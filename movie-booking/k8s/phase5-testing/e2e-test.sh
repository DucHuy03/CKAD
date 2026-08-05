#!/bin/bash
# e2e-test.sh - chay TOAN BO luong nghiep vu that qua api-gateway (NodePort
# 30080), y het flow da test o docker-compose (Phase 1) nhung lan nay chay
# tren k8s that. Neu script nay PASS het, coi nhu Phase 5 hoan tat.
#
# Yeu cau: da chay smoke-test.sh va PASS het, da cai jq.

set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:30080}"
PASS=0
FAIL=0

green() { printf "\033[32m%s\033[0m\n" "$1"; }
red()   { printf "\033[31m%s\033[0m\n" "$1"; }

step() { echo ""; echo "--- $1 ---"; }

check() {
  local desc="$1"
  local result="$2"
  if [ "$result" = "0" ]; then
    green "  [PASS] $desc"
    PASS=$((PASS+1))
  else
    red "  [FAIL] $desc"
    FAIL=$((FAIL+1))
  fi
}

step "1. Health check api-gateway"
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")
[ "$HEALTH" = "200" ]
check "GET /healthz tra ve 200" $?

step "2. Login lay JWT token"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"e2e-test-user","password":"anything"}')
TOKEN=$(echo "$LOGIN_RESP" | jq -r '.token // empty')
[ -n "$TOKEN" ]
check "nhan duoc JWT token" $?

step "3. Truy cap route can auth MA KHONG co token -> phai bi 401"
NO_AUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/movies" \
  -H "Content-Type: application/json" -d '{"title":"test"}')
[ "$NO_AUTH_CODE" = "401" ]
check "POST /api/movies khong token tra ve 401" $?

step "4. Tao cinema (can token)"
CINEMA_RESP=$(curl -s -X POST "$BASE_URL/api/cinemas" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"E2E Test Cinema","address":"Test St","city":"Test City"}')
CINEMA_ID=$(echo "$CINEMA_RESP" | jq -r '.id // empty')
[ -n "$CINEMA_ID" ]
check "tao cinema thanh cong, id=$CINEMA_ID" $?

step "5. Tao movie (can token)"
MOVIE_RESP=$(curl -s -X POST "$BASE_URL/api/movies" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"title":"E2E Test Movie","description":"test","duration_minutes":120,"genre":"Test"}')
MOVIE_ID=$(echo "$MOVIE_RESP" | jq -r '.id // empty')
[ -n "$MOVIE_ID" ]
check "tao movie thanh cong, id=$MOVIE_ID" $?

step "6. Tao showtime (can token)"
SHOWTIME_RESP=$(curl -s -X POST "$BASE_URL/api/showtimes" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"movie_id\":\"$MOVIE_ID\",\"cinema_id\":\"$CINEMA_ID\",\"room_name\":\"E2E Room\",\"start_time\":\"2026-08-01T19:00:00Z\",\"end_time\":\"2026-08-01T21:00:00Z\",\"price\":100000,\"total_seats\":50}")
SHOWTIME_ID=$(echo "$SHOWTIME_RESP" | jq -r '.id // empty')
[ -n "$SHOWTIME_ID" ]
check "tao showtime thanh cong, id=$SHOWTIME_ID" $?

step "7. Xem phim CONG KHAI (khong can token)"
PUBLIC_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/movies")
[ "$PUBLIC_CODE" = "200" ]
check "GET /api/movies khong token van tra ve 200" $?

SEAT="E2E$RANDOM" # ten ghe ngau nhien, tranh dung lai ghe da hold tu lan chay truoc
step "8. Hold ghe '$SEAT' (can token)"
HOLD_RESP=$(curl -s -X POST "$BASE_URL/api/bookings/hold" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"showtime_id\":\"$SHOWTIME_ID\",\"user_id\":\"e2e-test-user\",\"seats\":[\"$SEAT\"]}")
BOOKING_ID=$(echo "$HOLD_RESP" | jq -r '.id // empty')
[ -n "$BOOKING_ID" ]
check "hold ghe thanh cong, booking_id=$BOOKING_ID" $?

step "9. Kiem tra so do ghe - '$SEAT' phai nam trong 'held'"
SEATMAP=$(curl -s "$BASE_URL/api/showtimes/$SHOWTIME_ID/seats")
echo "$SEATMAP" | jq -e --arg s "$SEAT" '.held | index($s) != null' > /dev/null
check "ghe $SEAT xuat hien trong danh sach held" $?

step "10. Thanh toan (dung method mac dinh, ~90% thanh cong ngau nhien, thu toi da 3 lan)"
PAYMENT_OK=1
for i in 1 2 3; do
  PAYMENT_RESP=$(curl -s -X POST "$BASE_URL/api/payments/process" \
    -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d "{\"booking_id\":\"$BOOKING_ID\",\"showtime_id\":\"$SHOWTIME_ID\",\"user_id\":\"e2e-test-user\",\"amount\":100000,\"method\":\"CARD\",\"seats\":[\"$SEAT\"]}")
  PSTATUS=$(echo "$PAYMENT_RESP" | jq -r '.status // empty')
  if [ "$PSTATUS" = "SUCCESS" ]; then
    PAYMENT_OK=0
    break
  fi
  echo "    (lan $i: status=$PSTATUS, thu lai...)"
done
check "thanh toan SUCCESS trong toi da 3 lan thu" $PAYMENT_OK

step "11. Xac nhan booking chuyen sang CONFIRMED"
sleep 1
BOOKING_STATUS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/bookings/$BOOKING_ID" | jq -r '.status // empty')
[ "$BOOKING_STATUS" = "CONFIRMED" ]
check "booking status = CONFIRMED (thuc te: $BOOKING_STATUS)" $?

step "12. Xac nhan notification da duoc xu ly (cho consumer RabbitMQ xu ly xong)"
NOTIF_STATUS=""
for i in 1 2 3 4 5; do
  NOTIF_STATUS=$(curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/notifications/by-booking/$BOOKING_ID" | jq -r '.status // empty')
  [ "$NOTIF_STATUS" = "SENT" ] && break
  sleep 2
done
[ "$NOTIF_STATUS" = "SENT" ]
check "notification status = SENT (thuc te: $NOTIF_STATUS)" $?

echo ""
echo "================================"
green "PASS: $PASS"
[ "$FAIL" -gt 0 ] && red "FAIL: $FAIL" || echo "FAIL: $FAIL"
echo "================================"

if [ "$FAIL" -eq 0 ]; then
  echo ""
  green "TOAN BO FLOW NGHIEP VU CHAY DUNG TREN K8S - PHASE 5 HOAN TAT"
  echo "Kiem tra them bang mat: mo http://localhost:8025 (sau khi port-forward mailhog) xem email that"
fi

exit $FAIL