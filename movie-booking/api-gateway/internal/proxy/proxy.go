// Package proxy dựng reverse proxy chuyển tiếp request từ api-gateway sang
// từng backend service - đây là bản chất "gateway" của service này: nó
// không tự xử lý nghiệp vụ, chỉ định tuyến request.
package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// New tạo 1 httputil.ReverseProxy trỏ tới targetBase (vd "http://movie-service:8080").
//
// TỰ VIẾT Director thay vì dùng httputil.NewSingleHostReverseProxy(target)
// có sẵn, vì cần thêm 1 bước: XOÁ tiền tố "/api" khỏi đường dẫn trước khi
// forward. Client gọi api-gateway ở "/api/movies/123", nhưng movie-service
// thực sự chỉ hiểu route "/movies/123" (không có "/api") - "/api" chỉ là
// quy ước ở tầng gateway để phân biệt "đây là gọi qua gateway" với các port
// nội bộ khác, backend service không cần biết tới tiền tố này.
func New(targetBase string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetBase)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api")
			// Ghi de Host header thanh host cua backend - mot so framework/
			// middleware o backend dua vao Host header de routing/logging,
			// neu khong ghi de se van giu Host header goc (ten mien cua
			// api-gateway) gay nham lan.
			req.Host = target.Host
		},
		// ErrorHandler bat cac loi tang network (backend khong phan hoi,
		// connection refused...) - neu khong co, client se chi thay 1
		// connection bi dong dot ngot, kho debug hon nhieu so voi 1 JSON
		// 502 ro rang kem log phia gateway.
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("loi proxy toi %s: %v", targetBase, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"backend service khong phan hoi"}`))
		},
	}

	return proxy, nil
}
