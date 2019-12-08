// Reference: https://github.com/prometheus/prometheus/blob/master/scrape/scrape.go
package prom

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/k1LoW/mackerel-plugin-prometheus/version"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
)

const DefaultPrefix = "prom"

var replaceRe = regexp.MustCompile(`[\s{},\/"=]`)

type PrometheusPlugin struct {
	prefix  string
	targets []string
	graphs  map[string]mp.Graphs
	metrics map[string]float64
	client  *http.Client
}

// NewPrometheusPlugin returns PrometheusPlugin
func NewPrometheusPlugin(ctx context.Context, targets []string, prefix string) (PrometheusPlugin, error) {
	if prefix == "" {
		prefix = DefaultPrefix
	}

	p := PrometheusPlugin{
		targets: targets,
		graphs:  map[string]mp.Graphs{},
		metrics: map[string]float64{},
		prefix:  prefix,
		client:  newClient(),
	}

	mutex := new(sync.Mutex)

	for _, t := range targets {
		var buf = new(bytes.Buffer)
		_, err := p.scrape(ctx, t, buf)
		if err != nil {
			return p, err
		}

		parser := textparse.NewPromParser(buf.Bytes())

		var res labels.Labels

		for {
			et, err := parser.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return p, err
			}

			switch et {
			case textparse.EntrySeries:
				_, _, v := parser.Series()
				parser.Metric(&res)
				key := res.Get(labels.MetricName)

				b := labels.NewBuilder(res)
				b.Del(labels.MetricName)

				mutex.Lock()
				g, ok := p.graphs[key]
				if !ok {
					g = mp.Graphs{
						Label:   fmt.Sprintf("%s.%s", p.MetricKeyPrefix(), key),
						Unit:    mp.UnitFloat,
						Metrics: []mp.Metrics{},
					}
				}
				name := replaceRe.ReplaceAllString(fmt.Sprintf("%s-%s", key, b.Labels().String()), "")
				label := b.Labels().String()
				g.Metrics = append(g.Metrics, mp.Metrics{
					Name:    name,
					Label:   label,
					Diff:    false,
					Stacked: false,
				})
				p.graphs[key] = g
				p.metrics[name] = v
				mutex.Unlock()
				res = res[:0]

			case textparse.EntryType:
				// m, typ := parser.Type()
				// fmt.Printf("%v, %v\n", m, typ)

			case textparse.EntryHelp:
				// m, h := parser.Help()
				// fmt.Printf("%v, %v\n", m, h)

			case textparse.EntryComment:
				// fmt.Printf("%v\n", string(parser.Comment()))
			}
		}
	}

	return p, nil
}

func (p PrometheusPlugin) GraphDefinition() map[string]mp.Graphs {
	return p.graphs
}

func (p PrometheusPlugin) FetchMetrics() (map[string]float64, error) {
	return p.metrics, nil
}

func (p PrometheusPlugin) MetricKeyPrefix() string {
	return p.prefix
}

const acceptHeader = `application/openmetrics-text; version=0.0.1,text/plain;version=0.0.4;q=0.5,*/*;q=0.1`

var userAgentHeader = fmt.Sprintf("mackerel-plugin-prometheus/%s", version.Version)
var timeout = time.Duration(10 * time.Second)

func newClient() *http.Client {
	return &http.Client{}
}

func (p PrometheusPlugin) scrape(ctx context.Context, url string, w io.Writer) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", acceptHeader)
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", userAgentHeader)
	req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", fmt.Sprintf("%f", timeout.Seconds()))

	resp, err := p.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("server returned HTTP status %s", resp.Status)
	}

	if resp.Header.Get("Content-Encoding") != "gzip" {
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			return "", err
		}
		return resp.Header.Get("Content-Type"), nil
	}

	buf := bufio.NewReader(resp.Body)
	gzipr, err := gzip.NewReader(buf)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(w, gzipr)
	gzipr.Close()
	if err != nil {
		return "", err
	}
	return resp.Header.Get("Content-Type"), nil
}
