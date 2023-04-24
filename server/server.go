// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

// Server struct
type Server struct {
	// Database Connection
	db *sql.DB

	// Connections pool
	connections map[string]*websocket.Conn
}

// use default options
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Allow only my computer origin to connect to server for now
func checkOrigin(r *http.Request) bool {
	// Get the origin from the request
	origin := r.Header.Get("Origin")

	// Replace this with your logic to check the origin
	if origin != "http://localhost:3000" {
		log.Printf("Origin %s not allowed\n", origin)
		return false
	}

	return true
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// Socket connection
func (s Server) broadcast(w http.ResponseWriter, r *http.Request) {

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	//Generate random username
	username := RandStringRunes(5)

	//Insert user connection
	s.connections[username] = c

	defer c.Close()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		//Server message
		log.Printf("receive: %s", message)

		//statement := fmt.Sprintf("INSERT INTO messages(username,message,channelId) VALUES(\"%s\",\"%s\",1);", username, strings.TrimSpace(string(message)))
		_, err = s.db.Exec("INSERT INTO messages(username,message,channelId) VALUES(?,?,1);", username, strings.TrimSpace(string(message)))
		if err != nil {
			log.Printf("This is what happened %w", err)
		}

		for _, otherConn := range s.connections {
			if otherConn != c {
				err := otherConn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Failed to write message to connection: %s", err)
				}
			}
		}
	}

	//Removing the closed connection from connection pool
	delete(s.connections, username)

	c.Close()
}

// Database connection
const file string = "../Database/Messaging.db"

const users string = `
  CREATE TABLE IF NOT EXISTS users (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  username TEXT UNIQUE,
  description TEXT
  );`

const messages string = `
  CREATE TABLE IF NOT EXISTS messages (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  username TEXT,
  message TEXT,
  channelId INTEGER,
  FOREIGN KEY (channelId) REFERENCES channels(id),
  FOREIGN KEY (username) REFERENCES users(username)
  );`

const channels string = `
  CREATE TABLE IF NOT EXISTS channels (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  description TEXT
  );`

func CreateTables(db *sql.DB) {

	_, err := db.Exec(users)

	if nil != err {
		log.Printf("Failed to create users table %w", err)
		return
	}

	_, err = db.Exec(messages)

	if nil != err {
		log.Printf("Failed to create messages table%w", err)
		return
	}

	_, err = db.Exec(channels)

	if nil != err {
		log.Printf("Failed to create channels table %w", err)
		return
	}

}

// API endpoints
type Message struct {
	Username  string //`json:"username"`
	Message   string //`json:"message"`
	ChannelId int64  //`json:"channelId"`
}

func (s Server) getChannelMessages(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	rows, err := s.db.Query("SELECT username,message,channelId FROM messages WHERE channelId=?;", id)
	if nil != err {
		log.Printf("Failed to get channel messages%w", err)
		return
	}

	defer rows.Close()

	// An album slice to hold data from returned rows.
	var messages []Message

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var mesg Message
		if err := rows.Scan(&mesg.Username, &mesg.Message, &mesg.ChannelId); err != nil {
			log.Printf("Failed to get all messages %w", err)
			return
		}
		messages = append(messages, mesg)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Query failed %w", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func main() {
	db, err := sql.Open("sqlite3", file)

	if nil != err {
		log.Printf("This is what happened %w", err)
		return
	}

	defer db.Close()

	upgrader.CheckOrigin = checkOrigin

	s := Server{db: db, connections: make(map[string]*websocket.Conn)}
	CreateTables(db)
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/broadcast", s.broadcast)
	http.HandleFunc("/channel", s.getChannelMessages)
	log.Fatal(http.ListenAndServe(*addr, nil))

}
