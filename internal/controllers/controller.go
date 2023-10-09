package controllers

import (
	"asse-test/internal/msg"
	"asse-test/pkg/nettools"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
)

type Controller struct {
	msgChan          chan msg.Msg
	expectedMessages map[string][]msg.Msg //remoteAddr:msgs
	unprocessedMsg   map[string]int       //msgId:retries
	processedMsg     []msg.Msg
}

func NewController(msgChan chan msg.Msg) *Controller {
	return &Controller{
		msgChan:          msgChan,
		expectedMessages: make(map[string][]msg.Msg),
		unprocessedMsg:   make(map[string]int),
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

	for _, k := range c.expectedMessages[remoteAddr] {
		if slices.Contains(report, k.Id) {
			c.processedMsg = append(c.processedMsg, k)
		} else {
			if v, ok := c.unprocessedMsg[k.Id]; !ok {
				c.unprocessedMsg[k.Id] = 1
				c.msgChan <- k
			} else {
				c.unprocessedMsg[k.Id] += 1
				if v >= 3 {
					//add to db
				} else {
					c.msgChan <- k
				}
			}
		}
	}
}
