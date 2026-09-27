package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	h := spaHandler(fsys)

	tests := []struct {
		path, wantBody, wantCache string
	}{
		{"/", "app", "no-cache"},
		{"/tenants/123", "app", "no-cache"},
		{"/assets/app.js", "console.log", "immutable"},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", tt.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tt.wantBody) {
			t.Errorf("%s: body %q", tt.path, rec.Body.String())
		}
		if !strings.Contains(rec.Header().Get("Cache-Control"), tt.wantCache) {
			t.Errorf("%s: cache-control %q", tt.path, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestNotBuiltPlaceholder(t *testing.T) {
	rec := httptest.NewRecorder()
	spaHandler(fstest.MapFS{".keep": {}}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "make web") {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestEmbeddedHandlerServes(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", rec.Code)
	}
}
