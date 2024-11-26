package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	currDir, _ := os.Getwd()

	fmt.Println("Simple Go Shell")
	fmt.Println("------------------")

	for {
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if len(input) == 0 {
			continue
		}

		if input == "exit" {
			os.Exit(0)
		}

		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}
		cmdName := parts[0]
		args := parts[1:]

		cmd := exec.Command(cmdName, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Println(err)
		}
	}
}
