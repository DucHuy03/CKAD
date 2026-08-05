#!/bin/sh
# entrypoint.sh - logic cua sidecar log-shipper.
#
# FLOW:
#  1. Doc duong dan file log tu bien moi truong LOG_FILE_PATH (dung chung
#     gia tri voi container app trong cung Pod - ca 2 container tro toi
#     CUNG 1 duong dan tren volume emptyDir dung chung).
#  2. CHO cho toi khi file do xuat hien - vi thu tu start giua cac container
#     "app" va cac container khac (khong phai initContainer) trong CUNG Pod
#     la KHONG DAM BAO (co the log-shipper start truoc ca app container),
#     neu tail ngay ma file chua ton tai se bao loi va thoat.
#  3. tail -F: vua "follow" (theo doi noi dung moi duoc ghi them), vua tu
#     dong retry neu file bi xoa/tao lai (vd app container restart va mo
#     lai file) - phu hop hon "tail -f" thong thuong (khong retry duoc).

set -eu

LOG_FILE_PATH="${LOG_FILE_PATH:-/var/log/app/app.log}"

echo "log-shipper: dang cho file log xuat hien tai ${LOG_FILE_PATH}"

while [ ! -f "$LOG_FILE_PATH" ]; do
  sleep 1
done

echo "log-shipper: da tim thay file, bat dau tail (Ctrl+C hoac container stop de dung)"

# exec: thay the process shell hien tai bang tail, de tail nhan dung tin
# hieu SIGTERM tu k8s khi Pod bi xoa (khong bi shell "nuot" mat tin hieu),
# giup container dung nhanh gon thay vi phai cho het grace period.
exec tail -F -n +1 "$LOG_FILE_PATH"