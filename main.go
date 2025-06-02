package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Monitor peak memory usage of a command, similar to 'time' but for memory.\n")
		os.Exit(1)
	}

	// Parse command and arguments
	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	// Start the command
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the process
	err := cmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mempeak: failed to start command: %v\n", err)
		os.Exit(1)
	}

	pid := cmd.Process.Pid
	var peakMemory uint64

	// Monitor memory usage in a separate goroutine
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond) // Poll every 100ms
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				memory := getMemoryUsage(pid)
				if memory > peakMemory {
					peakMemory = memory
				}
			}
		}
	}()

	// Wait for the command to complete
	err = cmd.Wait()
	done <- true

	// Get the exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			fmt.Fprintf(os.Stderr, "mempeak: command failed: %v\n", err)
			exitCode = 1
		}
	}

	// Format and print the peak memory usage
	fmt.Fprintf(os.Stderr, "mempeak: peak memory usage: %s\n", formatBytes(peakMemory))

	// Exit with the same code as the monitored command
	os.Exit(exitCode)
}

// getMemoryUsage reads memory usage from /proc/pid/status on Linux
// or uses ps command on macOS
func getMemoryUsage(pid int) uint64 {
	// Try Linux /proc filesystem first
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	if data, err := os.ReadFile(statusFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						return kb * 1024 // Convert KB to bytes
					}
				}
			}
		}
	}

	// Fall back to ps command (works on macOS and other Unix systems)
	cmd := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	rssStr := strings.TrimSpace(string(output))
	if rss, err := strconv.ParseUint(rssStr, 10, 64); err == nil {
		return rss * 1024 // ps reports in KB, convert to bytes
	}

	return 0
}

// formatBytes formats bytes in human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}
