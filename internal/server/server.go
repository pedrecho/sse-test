package server

import (
	"asse-test/internal/controllers"
	"asse-test/internal/msg"
	"net/http"
)

func Run() error {

	controller := controllers.NewController(msg.NewMsgChan())

	http.HandleFunc("/report", controller.Report)

	http.ListenAndServe(":8080", nil)
	return nil
}
