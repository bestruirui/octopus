package relay

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	maxRelayJSONBodyBytes      int64 = 64 << 20
	maxRelayMultipartBodyBytes int64 = 64 << 20

	errRelayRequestBodyTooLarge = errors.New("request body too large")
)

func readLimitedRequestBody(c *gin.Context, limit int64) ([]byte, error) {
	limitRequestBody(c, limit)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, normalizeRelayRequestBodyError(err)
	}
	return body, nil
}

func limitRequestBody(c *gin.Context, limit int64) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
}

func normalizeRelayRequestBodyError(err error) error {
	if err == nil {
		return nil
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return errRelayRequestBodyTooLarge
	}
	return err
}

func relayRequestBodyErrorStatus(err error) int {
	if errors.Is(err, errRelayRequestBodyTooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}
