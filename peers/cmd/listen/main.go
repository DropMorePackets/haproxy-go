package main

import (
	"errors"
	"io"
	"log"
	"net"
	"strings"

	"github.com/fionera/haproxy-go/peers"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	listener, err := peers.Listen("tcp", ":21000", &peers.Config{
		UpdateHandler: func(update *peers.EntryUpdate) {
			log.Println(update.String())
		},
	})
	if err != nil {
		panic(err)
	}

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
			if errors.Is(err, net.ErrClosed) {
				log.Println("closed")
				return
			}
			if errors.Is(err, io.EOF) {
				log.Println("eof")
				return
			}

			if strings.Contains(err.Error(), "not implemented") {
				panic(err)
			}

			log.Println(err)
		}
	}
}
