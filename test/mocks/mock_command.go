package mocks

import (
	"io"
	"sync"
)

// MockCommandExecutor implements interfaces.CommandExecutor for testing
type MockCommandExecutor struct {
	mu                sync.RWMutex
	executedCommands  []string
	commandResults    map[string]bool
	commandOutputs    map[string]string
	defaultResult     bool
	defaultStdout     string
	defaultStderr     string
}

// NewMockCommandExecutor creates a new MockCommandExecutor
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		executedCommands: make([]string, 0),
		commandResults:   make(map[string]bool),
		commandOutputs:   make(map[string]string),
		defaultResult:    true,
	}
}

// SetCommandResult sets the result for a specific command
func (e *MockCommandExecutor) SetCommandResult(command string, result bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.commandResults[command] = result
}

// SetCommandOutput sets the output for a specific command
func (e *MockCommandExecutor) SetCommandOutput(command string, output string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.commandOutputs[command] = output
}

// SetDefaultResult sets the default result for commands
func (e *MockCommandExecutor) SetDefaultResult(result bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultResult = result
}

// SetDefaultOutput sets the default output for commands
func (e *MockCommandExecutor) SetDefaultOutput(stdout, stderr string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultStdout = stdout
	e.defaultStderr = stderr
}

// GetExecutedCommands returns the list of executed commands
func (e *MockCommandExecutor) GetExecutedCommands() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]string{}, e.executedCommands...)
}

// Execute executes a command
func (e *MockCommandExecutor) Execute(commandLine string, stdout, stderr io.Writer) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Record the command
	e.executedCommands = append(e.executedCommands, commandLine)
	
	// Get the result
	result, exists := e.commandResults[commandLine]
	if !exists {
		result = e.defaultResult
	}
	
	// Get the output
	output, exists := e.commandOutputs[commandLine]
	if !exists {
		output = e.defaultStdout
	}
	
	// Write the output
	if stdout != nil && output != "" {
		stdout.Write([]byte(output))
	}
	
	if stderr != nil && !result && e.defaultStderr != "" {
		stderr.Write([]byte(e.defaultStderr))
	}
	
	return result
}
