package influxdb

import (
	"context"
	"github.com/influxdata/influxdb-client-go"
	"github.com/vitelabs/go-vite/metrics"
	"testing"
	"time"
)

const (
	connection   = "http://127.0.0.1:9999"
	token        = "metrics"
	username     = "vite"
	userpassword = "vite@cool"
	bucket       = "vite-bucket"
	org          = "vite-org"
	namespace    = "monitor"
)

func TestReportToInfluxDB(t *testing.T) {
	metrics.InitMetrics(true, true)
	go metrics.CollectProcessMetrics(3 * time.Second)

	/*	go ReportToInfluxDB(metrics.DefaultRegistry, 10*time.Second,
			connection, token, username, userpassword, namespace, bucket, org,
			map[string]string{"host": "localhost"})*/

}

func TestInfluxDBV2(t *testing.T) {
	influx, err := influxdb.New(connection, token, influxdb.WithUserAndPass(username, userpassword))
	if err != nil {
		t.Fatal(err)
	}

	err = influx.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}
