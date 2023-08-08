package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fionera/haproxy-go/peers"
)

var metric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "haproxy",
	Subsystem: "stick_table",
	Name:      "data",
	Help:      "",
}, []string{"table", "type", "key"})

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	listener, err := peers.Listen("tcp", ":21000", &peers.Config{
		UpdateHandler: func(update *peers.EntryUpdate) {
			for i, d := range update.Data {
				dt := update.StickTable.DataTypes[i].DataType
				name := peers.StickTableDataTypes[dt].Name
				switch d.(type) {
				case *peers.FreqData:
					v := d.(*peers.FreqData)
					metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(v.LastPeriod))
				case *peers.SignedIntegerData:
					v := d.(*peers.SignedIntegerData)
					metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(*v))
				case *peers.UnsignedIntegerData:
					v := d.(*peers.UnsignedIntegerData)
					metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(*v))
				case *peers.UnsignedLongLongData:
					v := d.(*peers.UnsignedLongLongData)
					metric.WithLabelValues(update.StickTable.Name, name, update.Key.String()).Set(float64(*v))
				}
			}
		},
	})
	if err != nil {
		panic(err)
	}

	go http.ListenAndServe(":8081", promhttp.Handler())

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go connHandler(conn)
	}
}

func connHandler(conn *peers.Conn) {
	err := conn.Handshake()
	if err != nil {
		log.Fatal(err)
	}

	for {
		err := conn.Read()
		if err != nil {
			log.Println(err)
			conn.Close()
			return
		}
	}
}
