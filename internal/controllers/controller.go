package controllers

import (
	"asse-test/internal/msg"
	"asse-test/pkg/nettools"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"sync"
)

type Controller struct {
	msgChan          chan msg.Msg
	expectedMessages map[string][]msg.Msg //remoteAddr:msgs
	unprocessedMsg   map[string]int       //msgId:retries
	processedMsg     []msg.Msg
	db               *bolt.DB
	mu               sync.Mutex
}

func NewController(msgChan chan msg.Msg) (*Controller, error) {
	db, err := bolt.Open("mydb.db", 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Controller{
		msgChan:          msgChan,
		expectedMessages: make(map[string][]msg.Msg),
		unprocessedMsg:   make(map[string]int),
		db:               db,
		mu:               sync.Mutex{},
	}, nil
}

func (c *Controller) Shutdown() {
	c.db.Close()
}

func (c *Controller) SseHandler(w http.ResponseWriter, r *http.Request) {

	remoteAddr := nettools.ReadUserIP(r)

	slog.With("RemoteAddr", remoteAddr).Info("connected")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	defer func() {
		slog.With("RemoteAddr", remoteAddr).Info("connection closed")
	}()

	batchSize, err := strconv.Atoi(r.URL.Query().Get("batchsize"))
	if err != nil {
		slog.With("RemoteAddr", remoteAddr).Info("Wrong batchsize value")
		w.Write([]byte("Wrong batchsize value"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.With("RemoteAddr", remoteAddr).Error("Could not init http.Flusher")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for i := 0; i < batchSize; i++ {
		select {
		case message := <-c.msgChan:
			c.mu.Lock()
			c.expectedMessages[remoteAddr] = append(c.expectedMessages[remoteAddr], message)
			c.mu.Unlock()
			slog.With("RemoteAddr", remoteAddr).Info(fmt.Sprintf("Send message id: %s", message.Id))
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				slog.With("RemoteAddr", remoteAddr).Error("Could not marshal message")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "%s\n\n", string(jsonMessage))
			flusher.Flush()
		case <-r.Context().Done():
			slog.With("RemoteAddr", remoteAddr).Info("Client closed connection")
			return
		}
	}
}

func (c *Controller) Report(w http.ResponseWriter, r *http.Request) {

	remoteAddr := nettools.ReadUserIP(r)

	var report []string
	err := json.NewDecoder(r.Body).Decode(&report)
	if err != nil {
		slog.With("RemoteAddr", remoteAddr).Info("Wrong report data")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slog.With("RemoteAddr", remoteAddr).Info("Received report from")
	if _, ok := c.expectedMessages[remoteAddr]; !ok {
		slog.With("RemoteAddr", remoteAddr).Info("Unexpected connection")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() {
		c.mu.Lock()
		c.expectedMessages[remoteAddr] = c.expectedMessages[remoteAddr][:1]
		c.mu.Unlock()
	}()
	for _, k := range c.expectedMessages[remoteAddr] {
		if slices.Contains(report, k.Id) {
			c.mu.Lock()
			c.processedMsg = append(c.processedMsg, k)
			c.mu.Unlock()
		} else {
			if v, ok := c.unprocessedMsg[k.Id]; !ok {
				c.mu.Lock()
				c.unprocessedMsg[k.Id] = 1
				c.mu.Unlock()
				c.msgChan <- k
				slog.Info(fmt.Sprintf("Resent message %s", k.Id))
			} else {
				c.mu.Lock()
				c.unprocessedMsg[k.Id] += 1
				c.mu.Unlock()
				if v >= 3 {
					slog.Info(fmt.Sprintf("Message %s is unsuccessful", k.Id))
					err = c.db.Update(func(tx *bolt.Tx) error {
						bucket, err := tx.CreateBucketIfNotExists([]byte("people"))
						if err != nil {
							return err
						}
						data, err := json.Marshal(&k)
						if err != nil {
							return err
						}
						err = bucket.Put([]byte(k.Id), data)
						if err != nil {
							return err
						}
						return nil
					})
					if err != nil {
						slog.Error(fmt.Sprintf("Adding message to bbolt db: %v", err))
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					slog.Info(fmt.Sprintf("Message %s added to bbolt db", k.Id))
				} else {
					c.msgChan <- k
					slog.Info(fmt.Sprintf("Resent message %s", k.Id))
				}
			}
		}
	}
}
