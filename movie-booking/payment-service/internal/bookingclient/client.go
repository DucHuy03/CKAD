// Package bookingclient gọi HTTP sang booking-service - tương tự
// movieclient bên booking-service (xem comment ở đó để hiểu lý do dùng
// base URL cấu hình được thay vì hard-code).
package bookingclient

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// ConfirmBooking gọi POST {baseURL}/bookings/{id}/confirm.
// Đây là bước "chốt hạ" của cả luồng nghiệp vụ: chỉ sau khi payment-service
// xác nhận thanh toán thành công, booking mới chính thức chuyển từ HOLD
// sang CONFIRMED (ghế được đánh dấu bán vĩnh viễn ở booking-service).
func (c *Client) ConfirmBooking(ctx context.Context, bookingID string) error {
	url := fmt.Sprintf("%s/bookings/%s/confirm", c.baseURL, bookingID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("tao request confirm booking that bai: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("goi booking-service that bai: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("booking-service tra ve status khong mong doi khi confirm: %d", resp.StatusCode)
	}

	return nil
}
