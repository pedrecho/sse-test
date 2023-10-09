package msg

import (
	"github.com/rs/xid"
	"math/rand"
)

type Msg struct {
	Id     string `json:"id"`     // https://github.com/rs/xid
	Period uint64 `json:"period"` // 1-1000 random
}

func newMsg() Msg {
	return Msg{
		Id:     xid.New().String(),
		Period: rand.Uint64() % 1000,
	}
}
