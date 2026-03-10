package main

import (
	"Products_backend/cmd"
	"fmt"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Printf("Запуск программы не сработал!!!\n%v", err)
	}
}
