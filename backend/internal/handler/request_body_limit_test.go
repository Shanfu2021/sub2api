package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestRequestBodyLimitTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limit := int64(16)
	router := gin.New()
	router.Use(middleware.RequestBodyLimit(limit))
	router.POST("/test", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			if maxErr, ok := extractMaxBytesError(err); ok {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": buildBodyTooLargeMessage(maxErr.Limit),
				})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "read_failed",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	payload := bytes.Repeat([]byte("a"), int(limit+1))
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Contains(t, recorder.Body.String(), buildBodyTooLargeMessage(limit))
}

func TestGatewaySplitHandlersRequestBodyTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const limit int64 = 64
	cfg := &config.Config{}
	cfg.Gateway.MaxBodySize = limit
	gatewayHandler := &GatewayHandler{cfg: cfg}

	router := gin.New()
	router.Use(middleware.RequestBodyLimit(limit))
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{ID: 1})
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		c.Next()
	})
	router.POST("/v1/chat/completions", gatewayHandler.ChatCompletions)
	router.POST("/v1/responses", gatewayHandler.Responses)

	tests := []struct {
		name string
		path string
	}{
		{name: "chat_completions", path: "/v1/chat/completions"},
		{name: "responses", path: "/v1/responses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := bytes.Repeat([]byte("a"), int(limit+1))
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(payload))
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
			require.Equal(t, "Request body too large, limit is 64B", gjson.GetBytes(recorder.Body.Bytes(), "error.message").String())
		})
	}
}
