package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dtroode/gophermart/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLog_Handle(t *testing.T) {
	type logStruct struct {
		URI      string  `json:"uri"`
		Method   string  `json:"method"`
		Status   int     `json:"status"`
		Duration float64 `json:"duration"`
		Size     int     `json:"size"`
	}

	buf := bytes.NewBuffer(nil)
	logger := logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(buf, nil)),
	}

	middleware := NewRequestLog(&logger)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)

		resp, err := bufio.NewReader(r.Body).ReadByte()
		require.NoError(t, err)
		w.Write([]byte{resp})
	})

	r := httptest.NewRequest(http.MethodPatch, "/test-uri-patch", bytes.NewReader([]byte("test body")))
	w := httptest.NewRecorder()

	middleware.Handle(h).ServeHTTP(w, r)

	var logEntry = &logStruct{}
	err := json.NewDecoder(buf).Decode(logEntry)
	require.NoError(t, err)

	assert.Equal(t, "/test-uri-patch", logEntry.URI)
	assert.Equal(t, http.MethodPatch, logEntry.Method)
	assert.Equal(t, http.StatusAccepted, logEntry.Status)
	assert.Equal(t, 1, logEntry.Size)
}
