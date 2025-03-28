package interfaces

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

type OutputHandler func(line string)

type CommandExecutor interface {
	Execute(commandLine string, stdout, stderr io.Writer) bool
	ExecuteWithCallback(commandLine string, stdoutHandler, stderrHandler OutputHandler) bool
}

type OsCommandExecutor struct{}

func (e *OsCommandExecutor) Execute(commandLine string, stdout, stderr io.Writer) bool {
	cmd := exec.Command("sh", "-c", commandLine)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	return err == nil
}

func (e *OsCommandExecutor) ExecuteWithCallback(commandLine string, stdoutHandler, stderrHandler OutputHandler) bool {
	cmd := exec.Command("sh", "-c", commandLine)

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

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if stdoutHandler != nil {
				stdoutHandler(line)
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if stderrHandler != nil {
				stderrHandler(line)
			}
		}
	}()

	err = cmd.Wait()
	return err == nil
}

func NewOsCommandExecutor() *OsCommandExecutor {
	return &OsCommandExecutor{}
}
