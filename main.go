package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ProcessInfo struct {
	PID        int
	Name       string
	PeakMemory uint64
}

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

	rootPid := cmd.Process.Pid
	processStats := make(map[int]*ProcessInfo)
	var statsMutex sync.RWMutex

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
				pids := getProcessTree(rootPid)
				statsMutex.Lock()
				for _, pid := range pids {
					memory := getMemoryUsage(pid)
					name := getProcessName(pid)
					if memory > 0 {
						if info, exists := processStats[pid]; exists {
							if memory > info.PeakMemory {
								info.PeakMemory = memory
							}
						} else {
							processStats[pid] = &ProcessInfo{
								PID:        pid,
								Name:       name,
								PeakMemory: memory,
							}
						}
					}
				}
				statsMutex.Unlock()
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

	// Calculate total memory and print results
	statsMutex.RLock()
	var totalPeakMemory uint64
	var processes []*ProcessInfo
	for _, info := range processStats {
		processes = append(processes, info)
		totalPeakMemory += info.PeakMemory
	}
	statsMutex.RUnlock()

	// Sort processes by PID for consistent output
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})

	// Print per-process breakdown
	fmt.Fprintf(os.Stderr, "mempeak: process tree memory usage:\n")
	for _, info := range processes {
		fmt.Fprintf(os.Stderr, "  PID %d (%s): %s\n", info.PID, info.Name, formatBytes(info.PeakMemory))
	}
	fmt.Fprintf(os.Stderr, "mempeak: total peak memory usage: %s\n", formatBytes(totalPeakMemory))

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

// getProcessTree returns all PIDs in the process tree starting from rootPid
func getProcessTree(rootPid int) []int {
	pids := []int{rootPid}

	// Get all children recursively
	children := getChildProcesses(rootPid)
	for _, child := range children {
		childTree := getProcessTree(child)
		pids = append(pids, childTree...)
	}

	return pids
}

// getChildProcesses returns direct child processes of the given PID
func getChildProcesses(parentPid int) []int {
	var children []int

	// Try Linux /proc filesystem first
	if entries, err := os.ReadDir("/proc"); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			if pid, err := strconv.Atoi(entry.Name()); err == nil {
				statusFile := fmt.Sprintf("/proc/%d/stat", pid)
				if data, err := os.ReadFile(statusFile); err == nil {
					fields := strings.Fields(string(data))
					if len(fields) >= 4 {
						if ppid, err := strconv.Atoi(fields[3]); err == nil && ppid == parentPid {
							children = append(children, pid)
						}
					}
				}
			}
		}
		return children
	}

	// Fall back to ps command for macOS/Unix
	cmd := exec.Command("ps", "-eo", "pid,ppid")
	output, err := cmd.Output()
	if err != nil {
		return children
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if pid, err := strconv.Atoi(fields[0]); err == nil {
				if ppid, err := strconv.Atoi(fields[1]); err == nil && ppid == parentPid {
					children = append(children, pid)
				}
			}
		}
	}

	return children
}

// getProcessName returns the process name for the given PID
func getProcessName(pid int) string {
	// Try Linux /proc filesystem first
	commFile := fmt.Sprintf("/proc/%d/comm", pid)
	if data, err := os.ReadFile(commFile); err == nil {
		return strings.TrimSpace(string(data))
	}

	// Fall back to ps command for macOS/Unix
	cmd := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	name := strings.TrimSpace(string(output))
	if name == "" {
		return "unknown"
	}
	return name
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
