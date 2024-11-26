package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

// command is a type for shell commands.
type command string

const (
	pwd  command = "pwd"
	cd   command = "cd"
	exit command = "exit"
)

func pwdFn(w *io.PipeWriter) {
	defer w.Close()
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(w, dir)
}

func killCmds(cmds []*exec.Cmd, sig os.Signal) {
	for _, cmd := range cmds {
		cmd.Process.Signal(sig)
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cmds []*exec.Cmd

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				close(sigChan)
				return
			case sig := <-sigChan:
				switch sig {
				case os.Interrupt, syscall.SIGTERM:
					fmt.Println("\nReceived interrupt signal (Ctrl+C)")
					killCmds(cmds, sig)
					fmt.Print("> ")
				}
			}
		}
	}(ctx)

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

		commands := strings.Split(input, "|")
		var output io.ReadCloser

		for _, cmd := range commands {
			cmd = strings.TrimSpace(cmd)
			parts := strings.Fields(cmd)
			if len(parts) == 0 {
				continue
			}

			cmdName := parts[0]
			args := parts[1:]

			switch command(cmdName) {
			case exit:
				os.Exit(0)

			case cd:
				var path string
				if len(args) > 0 {
					path = args[0]
				} else {
					path, _ = os.UserHomeDir()
				}
				if err := os.Chdir(path); err != nil {
					fmt.Printf("%v\n", err)
				}
				continue

			case pwd:
				pr, pw := io.Pipe()
				go pwdFn(pw)
				if len(commands) == 1 {
					io.Copy(os.Stdout, pr)
					continue
				}
				output = pr

			default:
				cmd := exec.Command(cmdName, args...)
				cmd.Stderr = os.Stderr

				if output != nil {
					cmd.Stdin = output
				}

				// Set process group for the command
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

				cmds = append(cmds, cmd)
				output, _ = cmd.StdoutPipe()
			}
		}

		if len(cmds) > 0 {
			cmds[len(cmds)-1].Stdout = os.Stdout
		}

		// Create a process group for all commands.
		for _, cmd := range cmds {
			cmd.Start()
		}

		for _, cmd := range cmds {
			if err := cmd.Wait(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					// Check if the exit was due to a signal.
					if exitErr.ProcessState.ExitCode() == -1 && exitErr.ProcessState.Exited() {
						continue
					} else {
						fmt.Printf("Command exited with non-zero status: %d\n", exitErr.ExitCode())
					}
					continue
				}

				if cmd.ProcessState.ExitCode() == -1 {
					fmt.Printf("Command not found: %s\n", cmd.Path)
				}
			}
		}

		cmds = nil
	}
}
