package msg

import (
	"time"
)

const (
	chanBufSize       = 10
	maxGenerationSize = 5
	sleepDuration     = 2 * time.Second
)

func NewMsgChan() chan Msg {
	msgChan := make(chan Msg, chanBufSize)

	go func() {
		for {
			if len(msgChan) < maxGenerationSize {
				msgChan <- newMsg()
			}
			time.Sleep(sleepDuration)
		}
	}()

	return msgChan
}
