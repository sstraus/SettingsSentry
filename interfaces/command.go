package interfaces

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type OutputHandler func(line string)

type CommandExecutor interface {
	Execute(commandLine string, stdout, stderr io.Writer) bool
	ExecuteWithCallback(commandLine string, stdoutHandler, stderrHandler OutputHandler) bool
}

type OsCommandExecutor struct{}

func (e *OsCommandExecutor) Execute(commandLine string, stdout, stderr io.Writer) bool {
	cmd := exec.Command("bash", "-c", commandLine)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	return err == nil
}

func (e *OsCommandExecutor) ExecuteWithCallback(commandLine string, stdoutHandler, stderrHandler OutputHandler) bool {
	cmd := exec.Command("bash", "-c", commandLine)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return false
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error creating stderr pipe: %v\n", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		return false
	}

	// Use WaitGroup to ensure stdout/stderr reading goroutines complete
	// before returning. This prevents:
	// - Output truncation
	// - Resource leaks
	// - Race conditions between goroutine completion and cmd.Wait()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if stdoutHandler != nil {
				stdoutHandler(line)
			}
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if stderrHandler != nil {
				stderrHandler(line)
			}
		}
	}()

	err = cmd.Wait()

	// Wait for both goroutines to finish reading all output
	wg.Wait()

	return err == nil
}

func NewOsCommandExecutor() *OsCommandExecutor {
	return &OsCommandExecutor{}
}
