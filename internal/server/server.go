package server

import (
	"asse-test/internal/controllers"
	"asse-test/internal/msg"
	"net/http"
)

func Run() error {

	controller, err := controllers.NewController(msg.NewMsgChan())
	if err != nil {
		return err
	}
	defer controller.Shutdown()

	http.HandleFunc("/task", controller.SseHandler)
	http.HandleFunc("/report", controller.Report)

	http.ListenAndServe(":8080", nil)
	return nil
}
