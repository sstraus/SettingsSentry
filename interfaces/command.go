package interfaces

import (
	"io"
	"os/exec"
	"bufio"
	"fmt"
)

// OutputHandler is a function that processes each line of output
type OutputHandler func(line string)

// CommandExecutor defines an interface for executing commands
type CommandExecutor interface {
	// Execute runs a command and returns whether it was successful
	Execute(commandLine string, stdout, stderr io.Writer) bool
	
	// ExecuteWithCallback runs a command and calls the provided handler for each line of output
	ExecuteWithCallback(commandLine string, stdoutHandler, stderrHandler OutputHandler) bool
}

// OsCommandExecutor is the concrete implementation of CommandExecutor using OS exec
type OsCommandExecutor struct{}

// Execute executes a command using os/exec
func (e *OsCommandExecutor) Execute(commandLine string, stdout, stderr io.Writer) bool {
	cmd := exec.Command("sh", "-c", commandLine)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	
	err := cmd.Run()
	return err == nil
}

// ExecuteWithCallback executes a command and calls the provided handler for each line of output
func (e *OsCommandExecutor) ExecuteWithCallback(commandLine string, stdoutHandler, stderrHandler OutputHandler) bool {
	cmd := exec.Command("sh", "-c", commandLine)
	
	// Create pipes for stdout and stderr
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
	
	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		return false
	}
	
	// Process stdout in a goroutine
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if stdoutHandler != nil {
				stdoutHandler(line)
			}
		}
	}()
	
	// Process stderr in a goroutine
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if stderrHandler != nil {
				stderrHandler(line)
			}
		}
	}()
	
	// Wait for the command to complete
	err = cmd.Wait()
	return err == nil
}

// NewOsCommandExecutor creates a new OsCommandExecutor
func NewOsCommandExecutor() *OsCommandExecutor {
	return &OsCommandExecutor{}
}
