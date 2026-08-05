# log-shipper (sidecar dùng chung cho cả 5 service)

Tail 1 file log và in ra stdout của chính container này — để
`kubectl logs <pod> -c log-shipper` xem log tách biệt khỏi container `app`.

## Build

```bash
cd shared-images/log-shipper
docker build -t log-shipper:latest .
```

## Test nhanh (không cần k8s, dùng Docker volume mô phỏng emptyDir)

```bash
# Tao 1 volume Docker giu vai tro "emptyDir"
docker volume create test-log-vol

# Chay log-shipper truoc, tro toi file chua ton tai - phai thay dong
# "dang cho file log xuat hien" va KHONG bao loi, cu the cho mai
docker run --rm -v test-log-vol:/var/log/app -e LOG_FILE_PATH=/var/log/app/app.log log-shipper:latest &

# O 1 terminal khac, gia lap container app ghi log vao CUNG volume do
docker run --rm -v test-log-vol:/var/log/app alpine sh -c \
  'for i in $(seq 1 5); do echo "dong log so $i" >> /var/log/app/app.log; sleep 1; done'
```

Terminal chạy `log-shipper` phải in ra 5 dòng log ngay khi container kia ghi vào.

## Biến môi trường

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `LOG_FILE_PATH` | `/var/log/app/app.log` | Đường dẫn file log cần tail — PHẢI khớp với `LOG_FILE_PATH` mà container `app` dùng, và cả 2 container phải mount CÙNG volume tại đường dẫn cha của file này |

## Dùng ở Phase 4 (k8s)

```yaml
containers:
  - name: app
    env:
      - name: LOG_FILE_PATH
        value: /var/log/app/app.log
    volumeMounts:
      - name: app-logs
        mountPath: /var/log/app
  - name: log-shipper
    image: log-shipper:latest
    env:
      - name: LOG_FILE_PATH
        value: /var/log/app/app.log
    volumeMounts:
      - name: app-logs
        mountPath: /var/log/app
volumes:
  - name: app-logs
    emptyDir: {}
```