package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dropmorepackets/haproxy-go/peers"
	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
)

var metric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "haproxy",
	Subsystem: "stick_table",
	Name:      "data",
	Help:      "",
}, []string{"table", "type", "key"})

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	go http.ListenAndServe(":8081", promhttp.Handler())

	err := peers.ListenAndServe(":21000", peers.HandlerFunc(func(update *sticktable.EntryUpdate) {
		for i, d := range update.Data {
			dt := update.StickTable.DataTypes[i].DataType
			switch d := d.(type) {
			case *sticktable.FreqData:
				metric.WithLabelValues(update.StickTable.Name, dt.String(), update.Key.String()).Set(float64(d.LastPeriod))
			case *sticktable.SignedIntegerData:
				metric.WithLabelValues(update.StickTable.Name, dt.String(), update.Key.String()).Set(float64(*d))
			case *sticktable.UnsignedIntegerData:
				metric.WithLabelValues(update.StickTable.Name, dt.String(), update.Key.String()).Set(float64(*d))
			case *sticktable.UnsignedLongLongData:
				metric.WithLabelValues(update.StickTable.Name, dt.String(), update.Key.String()).Set(float64(*d))
			}
		}
	}))

	if err != nil {
		log.Fatal(err)
	}
}
