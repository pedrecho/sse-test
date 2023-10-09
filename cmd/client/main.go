package main

import "asse-test/internal/client"

func main() {
	if err := client.Run(); err != nil {
		panic(err)
	}
}
