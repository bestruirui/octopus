package helper

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

func GetUrlDelay(httpClient *http.Client, url string, ctx context.Context) (int, error) {
	if httpClient == nil {
		return 0, errors.New("http client is nil")
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return 0, errors.New("url is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return int(time.Since(start).Milliseconds()), nil
}
