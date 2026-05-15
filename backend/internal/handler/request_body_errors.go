package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// markOpsSkipForClientBodyReadError marks an early request-body disconnect so
// ops_error_logs can skip recording it as an actionable platform error.
func markOpsSkipForClientBodyReadError(c *gin.Context, err error) bool {
	if !isClientBodyReadDisconnect(err) {
		return false
	}
	if c != nil {
		c.Set(service.OpsSkipErrorLogKey, true)
	}
	return true
}

func isClientBodyReadDisconnect(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) && netErr != nil {
		msg := strings.ToLower(strings.TrimSpace(netErr.Error()))
		if strings.Contains(msg, "unexpected eof") ||
			strings.Contains(msg, "connection reset by peer") ||
			strings.Contains(msg, "broken pipe") ||
			strings.Contains(msg, "an established connection was aborted") {
			return true
		}
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "unexpected eof") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "an established connection was aborted")
}
