// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"flag"
	"log"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/broadcast"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				return
			}
			log.Printf("receive: %s", message)
		}
	}()

	for {
		select {
		default:
			// Taking input from user
			reader := bufio.NewReader(os.Stdin)
			log.Print("Enter your message: ")
			text, _ := reader.ReadString('\n')

			err := c.WriteMessage(websocket.TextMessage, []byte("Message sent from client: "+text))
			if err != nil {
				return
			}
		}
	}
}
