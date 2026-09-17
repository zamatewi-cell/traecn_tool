package main

import (
	"fmt"
	"os"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
)

func main() {
	a, err := auth.LoadTokenFromStorage(auth.DefaultStoragePath())
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	os.WriteFile("scripts/current_token.txt", []byte(a.Token), 0644)
	fmt.Printf("Loaded token: len=%d, prefix=%s\n", len(a.Token), a.Token[:20])
}
