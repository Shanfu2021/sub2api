package handler

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsClientBodyReadDisconnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "eof", err: io.EOF, want: true},
		{name: "unexpected_eof", err: io.ErrUnexpectedEOF, want: true},
		{name: "context_canceled", err: context.Canceled, want: true},
		{name: "wrapped_unexpected_eof", err: errors.New("failed to read body: unexpected EOF"), want: true},
		{name: "connection_reset", err: errors.New("read tcp 127.0.0.1:1->127.0.0.1:2: read: connection reset by peer"), want: true},
		{name: "broken_pipe", err: errors.New("write tcp 127.0.0.1:1->127.0.0.1:2: write: broken pipe"), want: true},
		{name: "gzip_header", err: errors.New("decode Content-Encoding \"gzip\": gzip: invalid header"), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isClientBodyReadDisconnect(tt.err))
		})
	}
}

func TestMarkOpsSkipForClientBodyReadError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	require.True(t, markOpsSkipForClientBodyReadError(c, io.ErrUnexpectedEOF))
	v, ok := c.Get(service.OpsSkipErrorLogKey)
	require.True(t, ok)
	require.Equal(t, true, v)

	rec2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rec2)
	require.False(t, markOpsSkipForClientBodyReadError(c2, errors.New("gzip: invalid header")))
	_, ok = c2.Get(service.OpsSkipErrorLogKey)
	require.False(t, ok)
}
