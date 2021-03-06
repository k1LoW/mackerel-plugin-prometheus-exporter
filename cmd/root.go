/*
Copyright © 2019 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/k1LoW/mackerel-plugin-prometheus-exporter/prome"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/spf13/cobra"
)

var (
	targets  []string
	prefix   string
	tempfile string
)

var rootCmd = &cobra.Command{
	Use:   "mackerel-plugin-prometheus-exporter",
	Short: "Mackerel plugin for reading Prometheus exporter metrics",
	Long:  `Mackerel plugin for reading Prometheus exporter metrics.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		p, err := prome.NewPlugin(ctx, prome.NewHTTPClient(), targets, prefix)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		plugin := mp.NewMackerelPlugin(p)
		plugin.Tempfile = tempfile
		plugin.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringArrayVarP(&targets, "target", "t", []string{}, "Prometheus exporter endpoint")
	rootCmd.Flags().StringVarP(&prefix, "prefix", "p", prome.DefaultPrefix, "Metric key prefix")
	rootCmd.Flags().StringVarP(&tempfile, "tempfile", "", "", "Temp file name")
}
