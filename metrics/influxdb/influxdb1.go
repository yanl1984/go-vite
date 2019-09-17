package influxdb

/*
import (
	"context"
	"fmt"
	"github.com/influxdata/influxdb-client-go"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/metrics"
	"sync"
	"time"
)

const PingRetryCellingCount int = 10

func ReportToInfluxDB(r metrics.Registry, d time.Duration, url, token, username, password, namespace, bucket, org string, tags map[string]string) {
	if !metrics.InfluxDBExportEnable {
		log.Info("influxdb export disable")
		return
	}
	rep, err := NewMetricsReport(r, d, url, token, username, password, namespace, bucket, org, tags)
	if err != nil || rep == nil {
		log.Error("unable to make influxdb client", "err", err)
	}
	rep.Start()
}

type MetricsReport struct {
	client   *influxdb.Client
	interval time.Duration

	bucket string
	org    string

	namespace string
	reg       metrics.Registry
	tags      map[string]string

	cache map[string]int64

	breaker      chan struct{}
	stopListener chan struct{}

	status      int
	statusMutex sync.Mutex
}

func NewMetricsReport(
	r metrics.Registry, d time.Duration,
	url, token, username, password, namespace, bucket, org string,
	tags map[string]string) (*MetricsReport, error) {

	influx, err := influxdb.New(url, token, influxdb.WithUserAndPass(username, password))
	if err != nil {
		return nil, err
	}

	err = influx.Ping(context.Background())
	if err != nil {
		return nil, err
	}
	return &MetricsReport{
		client:       influx,
		bucket:       bucket,
		interval:     d,
		org:          org,
		reg:          r,
		tags:         tags,
		namespace:    namespace,
		cache:        make(map[string]int64),
		breaker:      make(chan struct{}),
		stopListener: make(chan struct{}),
		status:       0,
		statusMutex:  sync.Mutex{},
	}, nil
}

func (r *MetricsReport) Start() {
	common.Go(r.run)
}

func (r *MetricsReport) Stop() {
	log.Info("reporter be called to stop")

	r.statusMutex.Lock()
	defer r.statusMutex.Unlock()
	if r.status != Stop {
		r.breaker <- struct{}{}
		close(r.breaker)

		<-r.stopListener
		close(r.stopListener)

		r.client.Close()

		metrics.InfluxDBExportEnable = false
		r.status = Stop
	}

	log.Info("reporter stoped")
}

func (r *MetricsReport) run() {
	log.Info("export started")
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(r.interval)
	ctx, cancel := context.WithTimeout(context.Background(), r.interval)
	defer cancel()
	count := 0
LOOP:
	for {
		if PingRetryCellingCount < count {
			r.Stop()
		}
		select {
		case <-intervalTicker:
			r.send()
		case <-pingTicker:
			if err := r.client.Ping(ctx); err != nil {
				log.Info("Client.Ping() error", "err", err)
				count++
			}
		case <-r.breaker:
			log.Info("call breaker")
			break LOOP
		}
	}

	log.Info("call stopListener")
	r.stopListener <- struct{}{}

	log.Info("export ended")
}

func (r *MetricsReport) send() {

	metricsSending := make([]influxdb.Metric, 0)
	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()
		var err error
		switch metric := i.(type) {
		case metrics.Counter:
			v := metric.Count()
			l := r.cache[name]
			metricsSending = append(metricsSending, influxdb.NewRowMetric(
				map[string]interface{}{"value": v - l},
				fmt.Sprintf("%s%s.count", r.namespace, name),
				r.tags,
				now))
		case metrics.Gauge:
			ms := metric.Snapshot()
			metricsSending = append(metricsSending, influxdb.NewRowMetric(
				map[string]interface{}{"value": ms.Value()},
				fmt.Sprintf("%s%s.count", r.namespace, name),
				r.tags,
				now))
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			metricsSending = append(metricsSending, influxdb.NewRowMetric(
				map[string]interface{}{"value": ms.Value()},
				fmt.Sprintf("%s%s.count", r.namespace, name),
				r.tags,
				now))
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			metricsSending = append(metricsSending, influxdb.NewRowMetric(
				map[string]interface{}{
					"count":    ms.Count(),
					"mean":     ms.Mean(),
					"max":      ms.Max(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
				},
				fmt.Sprintf("%s%s.histogram", r.namespace, name),
				r.tags,
				now))
		case metrics.Meter:
			ms := metric.Snapshot()
			metricsSending = append(metricsSending, influxdb.NewRowMetric(
				map[string]interface{}{
					"count": ms.Count(),
					"m1":    ms.Rate1(),
					"m5":    ms.Rate5(),
					"m15":   ms.Rate15(),
					"mean":  ms.RateMean(),
				},
				fmt.Sprintf("%s%s.meter", r.namespace, name),
				r.tags,
				now))
		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			metricsSending = append(metricsSending, influxdb.NewRowMetric(
				map[string]interface{}{
					"count":    ms.Count(),
					"mean":     ms.Mean(),
					"max":      ms.Max(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
					"m1":       ms.Rate1(),
					"m5":       ms.Rate5(),
					"m15":      ms.Rate15(),
					"meanrate": ms.RateMean(),
				},
				fmt.Sprintf("%s%s.timer", r.namespace, name),
				r.tags,
				now))
		case metrics.ResettingTimer:
			t := metric.Snapshot()
			if len(t.Values()) > 0 {
				ps := t.Percentiles([]float64{50, 95, 99})
				val := t.Values()
				metricsSending = append(metricsSending, influxdb.NewRowMetric(
					map[string]interface{}{
						"count": len(val),
						"mean":  t.Mean(),
						"max":   val[len(val)-1],
						"min":   val[0],
						"p50":   ps[0],
						"p95":   ps[1],
						"p99":   ps[2],
					},
					fmt.Sprintf("%s%s.span", r.namespace, name), r.tags,
					now))
			}
		default:
			log.Info("Unknown metric type")
			return
		}
		if err != nil {
			log.Info("unexpected error. expected %v, actual %v", nil, err)
			return
		}
	})

	// The actual write..., this method can be called concurrently.
	if _, err := r.client.Write(context.Background(), r.bucket, r.org, metricsSending...); err != nil {
		log.Error(err.Error()) // as above use your own error handling here.
		return
	}
}
*/
