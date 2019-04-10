package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/auyer/go-httpstat"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

const UpdateInterval time.Duration = 3 * time.Second

func main() {

	urls := make(map[string][]string)           // map for settings.yaml load
	data, _ := ioutil.ReadFile("settings.yaml") // Loading settings
	yaml.Unmarshal([]byte(data), &urls)         // deserialization
	// fmt.Println(urls)

	respMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "response_info"}, []string{"url", "metric"},
	)
	for _, url := range urls["urls"] {
		check(url, respMetric)
	}

	r := prometheus.NewRegistry()
	r.MustRegister(respMetric)

	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)
	log.Fatal(http.ListenAndServe(":8092", nil))

}

func check(url string, respMetric *prometheus.GaugeVec) {
	ticker := time.NewTicker(UpdateInterval)

	go func() {
		for range ticker.C {
			// Create a new HTTP request
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Fatal(err)
			}

			// Create a httpstat powered context
			var result httpstat.Result
			ctx := httpstat.WithHTTPStat(req.Context(), &result)
			req = req.WithContext(ctx)

			// Send request by default HTTP client
			client := http.DefaultClient
			client.CloseIdleConnections()
			res, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
				log.Fatal(err)
			}

			res.Body.Close()
			timeEndBody := time.Now()
			var total = result.Total(timeEndBody)

			if res.TLS != nil {
				CertExpiryDate := res.TLS.PeerCertificates[0].NotAfter
				CertExpiryDaysLeft := int64(CertExpiryDate.Sub(time.Now()).Hours() / 24)
				log.Printf("CertExpiryDate: %s", CertExpiryDate.Format("2006-01-02 15:04:05"))
				log.Printf("CertExpiryDate: %d days", int(CertExpiryDaysLeft))
				respMetric.WithLabelValues(url, "CertExpiryDate").Set(float64(CertExpiryDaysLeft))
			}

			log.Printf("URL: %s", req.URL)
			log.Printf("Status Code: %d", int(res.StatusCode))
			log.Printf("DNS lookup: %d ms", int(result.DNSLookup/time.Millisecond))
			log.Printf("TCP connection: %d ms", int(result.TCPConnection/time.Millisecond))
			log.Printf("TLS handshake: %d ms", int(result.TLSHandshake/time.Millisecond))
			log.Printf("Server processing: %d ms", int(result.ServerProcessing/time.Millisecond))
			log.Printf("Content transfer: %d ms", int(total/time.Millisecond))

			respMetric.WithLabelValues(url, "stataus_code").Set(float64(res.StatusCode))
			respMetric.WithLabelValues(url, "dns_lookup").Set(float64(result.DNSLookup / time.Millisecond))
			respMetric.WithLabelValues(url, "tcp_connection").Set(float64(result.TCPConnection / time.Millisecond))
			respMetric.WithLabelValues(url, "tls_handshake").Set(float64(result.TLSHandshake / time.Millisecond))
			respMetric.WithLabelValues(url, "server_processing").Set(float64(result.ServerProcessing / time.Millisecond))
			respMetric.WithLabelValues(url, "total_time").Set(float64(total / time.Millisecond))
		}
	}()
}
