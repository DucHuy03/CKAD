# nginx-ambassador (ambassador dùng chung cho cả 5 service)

Proxy request từ `localhost:<LISTEN_PORT>` (trong cùng Pod) sang backend thật
(có thể là service khác, hoặc chính container `app` cùng Pod tuỳ cách bố trí
ở Phase 4) — app container không cần biết địa chỉ mạng thật của phía bên kia,
chỉ cần gọi `localhost`.

## Build

```bash
cd shared-images/nginx-ambassador
docker build -t nginx-ambassador:latest .
```

## Test nhanh (không cần k8s)

```bash
# Chay thu 1 backend don gian de proxy toi
docker run --rm -d --name test-backend -p 9000:80 \
  --entrypoint sh nginx:alpine -c 'echo "xin chao tu backend that" > /usr/share/nginx/html/index.html && nginx -g "daemon off;"'

# Chay ambassador, tro UPSTREAM ve container backend o tren qua network Docker
docker network create test-net 2>/dev/null || true
docker network connect test-net test-backend

docker run --rm -d --name test-ambassador --network test-net -p 8090:8090 \
  -e LISTEN_PORT=8090 -e UPSTREAM_HOST=test-backend -e UPSTREAM_PORT=80 \
  nginx-ambassador:latest

curl http://localhost:8090/
# phai thay "xin chao tu backend that"

curl http://localhost:8090/ambassador-healthz
# phai thay "ambassador ok"

docker rm -f test-backend test-ambassador
docker network rm test-net
```

## Biến môi trường

| Biến | Ý nghĩa |
|---|---|
| `LISTEN_PORT` | Cổng ambassador lắng nghe (phải >1024 vì chạy non-root) |
| `UPSTREAM_HOST` | Host của backend thật cần proxy tới |
| `UPSTREAM_PORT` | Port của backend thật |

## Dùng ở Phase 4 (k8s)

Mỗi service sẽ có `LISTEN_PORT`/`UPSTREAM_HOST`/`UPSTREAM_PORT` khác nhau,
set qua `env` trong Deployment — KHÔNG cần build lại image, chỉ cần đổi giá
trị biến môi trường:

```yaml
containers:
  - name: nginx-ambassador
    image: nginx-ambassador:latest
    env:
      - name: LISTEN_PORT
        value: "8090"
      - name: UPSTREAM_HOST
        value: "booking-service"   # ten Service trong k8s
      - name: UPSTREAM_PORT
        value: "8080"
    ports:
      - containerPort: 8090
```

Container `app` trong cùng Pod (ví dụ `payment-service` cần gọi
`booking-service`) lúc đó sẽ đổi biến `BOOKING_SERVICE_URL` từ
`http://booking-service:8080` (gọi thẳng Service, docker-compose hiện tại)
thành `http://localhost:8090` (qua ambassador cùng Pod) — đây chính là thay
đổi duy nhất cần làm ở business logic khi chuyển từ docker-compose sang k8s.