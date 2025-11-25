package op

import (
	"context"
	"time"
)

func InitCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := settingRefreshCache(ctx); err != nil {
		return err
	}
	if err := channelRefreshCache(ctx); err != nil {
		return err
	}
	if err := groupRefreshCache(ctx); err != nil {
		return err
	}
	if err := apiKeyRefreshCache(ctx); err != nil {
		return err
	}
	if err := llmModelRefreshCache(ctx); err != nil {
		return err
	}
	if err := statsRefreshCache(ctx); err != nil {
		return err
	}
	return nil
}

func SaveCache() error {
	if err := StatsSaveDB(); err != nil {
		return err
	}
	return nil
}
