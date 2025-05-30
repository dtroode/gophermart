package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompress(t *testing.T) {
	expectedBody := []byte("test body")

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		assert.Equal(t, expectedBody, body)
	})

	t.Run("encoded request", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		gw := gzip.NewWriter(buf)
		_, err := gw.Write(expectedBody)
		require.NoError(t, err)

		err = gw.Close()
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/", buf)
		r.Header.Set("Content-Encoding", "gzip")

		w := httptest.NewRecorder()

		Decompress(dummyHandler).ServeHTTP(w, r)
	})

	t.Run("plain request", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(expectedBody))
		w := httptest.NewRecorder()

		Decompress(dummyHandler).ServeHTTP(w, r)
	})
}
