package removebg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Processor removes image backgrounds. Implementations must return PNG bytes.
type Processor interface {
	Process(context.Context, []byte) ([]byte, error)
}

type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func New(apiKey, baseURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = "https://api.remove.bg/v1.0/removebg"
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{APIKey: apiKey, BaseURL: baseURL, HTTP: &http.Client{Timeout: timeout}}
}

func (c *Client) Process(ctx context.Context, image []byte) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image_file", "signature.png")
	if err != nil {
		return nil, err
	}
	if _, err = part.Write(image); err != nil {
		return nil, err
	}
	_ = writer.WriteField("size", "auto")
	_ = writer.WriteField("format", "png")
	_ = writer.WriteField("type", "graphic")
	if err = writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("remove.bg returned HTTP %d: %s", res.StatusCode, string(message))
	}
	return io.ReadAll(io.LimitReader(res.Body, 6*1024*1024))
}
