package main

import "asse-test/internal/server"

func main() {
	if err := server.Run(); err != nil {
		panic(err)
	}
}
