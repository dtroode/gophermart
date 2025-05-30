package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipReader struct {
	g *gzip.Reader
	r io.ReadCloser
}

func (gr *gzipReader) Close() error {
	if err := gr.r.Close(); err != nil {
		return err
	}
	return gr.g.Close()
}

func (gr *gzipReader) Read(p []byte) (n int, err error) {
	return gr.g.Read(p)
}

func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &gzipReader{
		g: reader,
		r: r,
	}, nil
}

func Decompress(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ce := r.Header.Get("Content-Encoding")
		if strings.Contains(ce, "gzip") {
			cr, err := newGzipReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				return
			}
			r.Body = cr
			cr.Close()
		}

		h.ServeHTTP(w, r)
	})
}
