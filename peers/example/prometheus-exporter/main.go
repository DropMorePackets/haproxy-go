package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dropmorepackets/haproxy-go/peers"
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

	err := peers.ListenAndServe(":21000", peers.HandlerFunc(func(update *peers.EntryUpdate) {
		for i, d := range update.Data {
			dt := update.StickTable.DataTypes[i].DataType
			name := peers.StickTableDataTypes[dt].Name
			switch d := d.(type) {
			case *peers.FreqData:
				metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(d.LastPeriod))
			case *peers.SignedIntegerData:
				metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(*d))
			case *peers.UnsignedIntegerData:
				metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(*d))
			case *peers.UnsignedLongLongData:
				metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(*d))
			}
		}
	}))

	if err != nil {
		log.Fatal(err)
	}
}
