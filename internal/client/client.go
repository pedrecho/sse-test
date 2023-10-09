package client

import (
	"asse-test/internal/msg"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	requestURL  = "localhost:8080/task"
	responseURL = "localhost:8080/report"
)

func Run() error {

	batchSize := rand.Intn(9) + 1
	slog.Info(fmt.Sprintf("Batchsize: %d", batchSize))

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s?batchsize=%d", requestURL, batchSize), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected status code: %d", resp.StatusCode)
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var report []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		slog.Info(fmt.Sprintf("Received SSE message: %s\n", line))
		wg.Add(1)
		go func() {
			defer wg.Done()
			var msg msg.Msg
			err := json.Unmarshal([]byte(line), &msg)
			if err != nil {
				panic(err)
			}
			if msg.Period > 900 {
				panic(fmt.Sprintf("Msg period: %d", msg.Period))
			}
			defer func() {
				if r := recover(); r != nil {
					slog.Info("panic recovered")
				}
			}()
			if msg.Period > 800 {
				panic(fmt.Sprintf("Msg period: %d", msg.Period))
			}
			time.Sleep(time.Duration(msg.Period) * time.Millisecond)
			mu.Lock()
			report = append(report, msg.Id)
			mu.Unlock()
		}()
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	wg.Wait()
	jsonData, err := json.Marshal(report)
	if err != nil {
		return err
	}
	slog.Info(string(jsonData))
	resp, err = http.Post(fmt.Sprintf("http://%s", responseURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		slog.Info("Запрос успешно выполнен")
	} else {
		slog.Error("Ошибка:", resp.Status)
	}
	return nil
}
