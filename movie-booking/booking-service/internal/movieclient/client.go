// Package movieclient là HTTP client GỌI SANG movie-service - lần giao tiếp
// service-to-service đầu tiên trong hệ thống.
//
// Ở Phase 4 (k8s), booking-service sẽ KHÔNG gọi thẳng movie-service qua
// tên Service của k8s, mà gọi qua "localhost" tới container ambassador
// nginx nằm chung Pod với nó - nginx đó mới là nơi thực sự proxy request
// ra ngoài tới movie-service. Code ở đây chỉ cần biết 1 base URL
// (MovieServiceURL trong config) - hôm nay base URL trỏ thẳng movie-service
// (docker-compose), sau này đổi thành "http://localhost:8090" (ambassador)
// mà KHÔNG PHẢI SỬA CODE, chỉ đổi biến môi trường.
package movieclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ShowtimeInfo chỉ khai báo lại đúng những field booking-service thực sự
// cần dùng từ response JSON của movie-service (Price, TotalSeats...) -
// không cần import toàn bộ struct Showtime của movie-service. Đây là thực
// hành phổ biến trong kiến trúc microservices: mỗi service tự định nghĩa
// "hợp đồng" (contract) tối thiểu nó cần từ service khác, không phụ thuộc
// trực tiếp vào struct nội bộ của service kia.
type ShowtimeInfo struct {
	ID         string  `json:"id"`
	MovieID    string  `json:"movie_id"`
	CinemaID   string  `json:"cinema_id"`
	Price      float64 `json:"price"`
	TotalSeats int     `json:"total_seats"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		// Set timeout ro rang: neu movie-service treo, booking-service
		// khong bi treo theo vo thoi han - fail nhanh con hon la doi mai mai.
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// GetShowtime gọi GET {baseURL}/showtimes/{id} và parse response JSON.
// Trả lỗi rõ ràng nếu showtime không tồn tại (404) hoặc movie-service
// không phản hồi được - để handler gọi hàm này biết cách trả lỗi hợp lý
// cho client (400 nếu showtime_id sai, 503 nếu movie-service đang down).
func (c *Client) GetShowtime(ctx context.Context, showtimeID string) (*ShowtimeInfo, error) {
	url := fmt.Sprintf("%s/showtimes/%s", c.baseURL, showtimeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("tao request that bai: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("goi movie-service that bai: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("showtime %q khong ton tai", showtimeID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("movie-service tra ve status khong mong doi: %d", resp.StatusCode)
	}

	var info ShowtimeInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("parse response tu movie-service that bai: %w", err)
	}

	return &info, nil
}
