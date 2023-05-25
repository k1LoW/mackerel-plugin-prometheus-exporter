package prome

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func TestNewPlugin(t *testing.T) {
	for _, td := range []struct {
		title    string
		use_gzip bool
	}{
		{
			title:    "plain",
			use_gzip: false,
		},
		{
			title:    "gzip",
			use_gzip: true,
		},
	} {
		t.Run(td.title, func(t *testing.T) {
			in := `# HELP test metrics
# 	TYPE test_metrics_seconds counter
test_metrics_seconds{role="a" } 4.9351e-05
test_metrics_seconds{role="b",group="d"} 8.3835e-05
test_metrics_seconds{ role="c", group="e"} 8.3835e-05

# HELP test more metrics
# 	TYPE test_more_metrics_bytes gauge
test_more_metrics_bytes{role="a" } 256.0`

			ts := newMockServer(in, td.use_gzip)
			targets := []string{ts.URL}
			prefix := ""
			ctx := context.Background()
			p, err := NewPlugin(ctx, NewHTTPClient(), targets, prefix)
			if err != nil {
				t.Fatal(err)
			}
			g := p.GraphDefinition()
			if len(g) != 2 {
				t.Errorf("got %v want %v", len(g), 2)
			}

			m, _ := p.FetchMetrics()
			if len(m) != 4 {
				t.Errorf("got %v want %v", len(m), 4)
			}
		})
	}
}

func TestOpenMetricsFormat(t *testing.T) {
	for _, td := range []struct {
		title    string
		use_gzip bool
	}{
		{
			title:    "plain",
			use_gzip: false,
		},
		{
			title:    "gzip",
			use_gzip: true,
		},
	} {
		t.Run(td.title, func(t *testing.T) {
			in := `# TYPE test_metrics_seconds unknown
# HELP test_metrics_seconds metrics
test_metrics_seconds{role="a"} 4.9351e-05
test_metrics_seconds{role="b",group="d"} 8.3835e-05
test_metrics_seconds{role="c",group="e"} 8.3835e-05
# TYPE test_more_metrics_bytes gauge
# HELP test_more_metrics test more metrics
test_more_metrics_bytes{role="a"} 256.0
# EOF`

			ts := newOpenMetricsMockServer(in, td.use_gzip)
			targets := []string{ts.URL}
			prefix := ""
			ctx := context.Background()
			p, err := NewPlugin(ctx, NewHTTPClient(), targets, prefix)
			if err != nil {
				t.Fatal(err)
			}
			g := p.GraphDefinition()
			if len(g) != 2 {
				t.Errorf("got %v want %v", len(g), 2)
			}

			m, _ := p.FetchMetrics()
			if len(m) != 4 {
				t.Errorf("got %v want %v", len(m), 4)
			}
		})
	}

}

func newMockServer(in string, use_gzip bool) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if use_gzip {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			w = gzipResponseWriter{Writer: gz, ResponseWriter: w}
		}
		fmt.Fprintf(w, "%s", in)
	})
	return httptest.NewServer(handler)
}

func newOpenMetricsMockServer(in string, use_gzip bool) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if use_gzip {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			w = gzipResponseWriter{Writer: gz, ResponseWriter: w}
		}
		w.Header().Set("Content-Type", "application/openmetrics-text; version=1.0.0; charset=utf-8")
		fmt.Fprintf(w, "%s", in)
	})
	return httptest.NewServer(handler)
}
