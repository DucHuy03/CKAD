# CKAD Labs

Mỗi lab có hai lệnh ổn định ngay trong thư mục của lab:

```bash
# Chạy demo, có thể thay đổi tài nguyên Kubernetes.
bash labs/lab-1.1/run.sh

# Kiểm tra cấu trúc/evidence, không chủ đích thay đổi cluster.
bash labs/lab-1.1/check.sh
```

Tất cả 20 lab từ `lab-1.1` đến `lab-5.4` theo cùng quy ước `run.sh` và
`check.sh`.

Đọc [PREREQUISITES.md](PREREQUISITES.md) trước khi chạy. Tài liệu đó nêu các
biến `export` cần có, phụ thuộc giữa các lab, tác động lên cluster, và cách
tránh đầy quota trong namespace `lab`.
