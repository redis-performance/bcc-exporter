package main

import (
	"bytes"
	"crypto/subtle"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	port     = flag.String("port", "8080", "Port to listen on")
	password = flag.String("password", "", "Password for basic authentication (optional)")
)

func main() {
	flag.Parse()

	// Set up handlers with optional authentication
	if *password != "" {
		http.HandleFunc("/debug/pprof/profile", basicAuth(handlePprof, *password))
		http.HandleFunc("/debug/folded/profile", basicAuth(handleFolded, *password))
	} else {
		http.HandleFunc("/debug/pprof/profile", handlePprof)
		http.HandleFunc("/debug/folded/profile", handleFolded)
	}

	addr := ":" + *port
	log.Printf("Listening on %s...", addr)
	if *password != "" {
		log.Println("Basic authentication enabled")
	}
	log.Fatal(http.ListenAndServe(addr, nil))
}

// basicAuth wraps a handler with basic authentication
func basicAuth(handler http.HandlerFunc, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="bcc-exporter"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}

func handlePprof(w http.ResponseWriter, r *http.Request) {
	runProfile(w, r, "pprof")
}

func handleFolded(w http.ResponseWriter, r *http.Request) {
	runProfile(w, r, "folded")
}

func runProfile(w http.ResponseWriter, r *http.Request, format string) {
	pid := r.URL.Query().Get("pid")
	seconds := r.URL.Query().Get("seconds")
	testMode := r.URL.Query().Get("test") == "true"

	if pid == "" || seconds == "" {
		http.Error(w, "Missing pid or seconds", http.StatusBadRequest)
		return
	}

	dur, err := strconv.Atoi(seconds)
	if err != nil || dur <= 0 || dur > 300 {
		http.Error(w, "Invalid seconds", http.StatusBadRequest)
		return
	}

	// Test mode - return mock data
	if testMode {
		mockData := generateMockProfile(pid, dur)
		if format == "pprof" {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else {
			w.Header().Set("Content-Type", "text/plain")
		}
		w.Write([]byte(mockData))
		return
	}

	// Validate PID exists
	if err := validatePID(pid); err != nil {
		http.Error(w, fmt.Sprintf("Invalid PID: %v", err), http.StatusBadRequest)
		return
	}

	// For pprof format, use perf record + pprof conversion
	if format == "pprof" {
		runPerfProfile(w, r, pid, dur)
	} else {
		// For folded format, keep the old BCC approach for now
		runBCCProfile(w, r, pid, dur)
	}
}

// validatePID checks if the given PID exists and is accessible
func validatePID(pid string) error {
	// Check if PID is a valid number
	if _, err := strconv.Atoi(pid); err != nil {
		return fmt.Errorf("invalid PID format: %s", pid)
	}

	// Check if /proc/<pid> exists
	procPath := fmt.Sprintf("/proc/%s", pid)
	if _, err := os.Stat(procPath); os.IsNotExist(err) {
		return fmt.Errorf("process with PID %s does not exist", pid)
	} else if err != nil {
		return fmt.Errorf("cannot access process %s: %v", pid, err)
	}

	return nil
}

// checkRequiredTools verifies that perf and pprof tools are available
func checkRequiredTools() error {
	// Check if perf is available
	if _, err := exec.LookPath("perf"); err != nil {
		return fmt.Errorf("perf tool not found: %v. Install with: sudo apt-get install linux-perf", err)
	}

	// Check if pprof is available
	if _, err := exec.LookPath("pprof"); err != nil {
		return fmt.Errorf("pprof tool not found: %v. Install with: go install github.com/google/pprof@latest", err)
	}

	return nil
}

// runPerfProfile executes perf record + pprof conversion and serves the binary pprof file
func runPerfProfile(w http.ResponseWriter, r *http.Request, pid string, duration int) {
	// Check if required tools are available
	if err := checkRequiredTools(); err != nil {
		http.Error(w, fmt.Sprintf("Required tools not available: %v", err), http.StatusInternalServerError)
		return
	}

	// Create temporary directory for this profiling session
	tempDir, err := os.MkdirTemp("", "bcc-exporter-")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create temp directory: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir) // Clean up when done

	perfDataPath := filepath.Join(tempDir, "perf.data")
	pprofPath := filepath.Join(tempDir, "profile.pb.gz")

	// Step 1: Run perf record
	log.Printf("Starting perf record for PID %s, duration %d seconds", pid, duration)
	perfCmd := exec.Command("perf", "record", "-g", "--pid", pid, "-F", "999", "-o", perfDataPath, "--", "sleep", fmt.Sprintf("%d", duration))

	var perfStderr bytes.Buffer
	perfCmd.Stderr = &perfStderr

	if err := perfCmd.Run(); err != nil {
		log.Printf("perf record failed: %v", err)
		log.Printf("perf stderr: %s", perfStderr.String())

		// Provide more specific error messages
		stderrStr := perfStderr.String()
		if strings.Contains(stderrStr, "Permission denied") {
			http.Error(w, "Permission denied: perf requires elevated privileges. Run with sudo or adjust perf_event_paranoid settings.", http.StatusForbidden)
		} else if strings.Contains(stderrStr, "No such process") {
			http.Error(w, fmt.Sprintf("Process with PID %s not found or exited during profiling", pid), http.StatusBadRequest)
		} else {
			http.Error(w, fmt.Sprintf("perf record failed: %v\nStderr: %s", err, stderrStr), http.StatusInternalServerError)
		}
		return
	}

	// Check if perf.data was created and has content
	if stat, err := os.Stat(perfDataPath); err != nil {
		http.Error(w, "perf.data file was not created", http.StatusInternalServerError)
		return
	} else if stat.Size() == 0 {
		http.Error(w, "perf.data file is empty - no samples collected", http.StatusInternalServerError)
		return
	}

	// Step 2: Convert perf.data to pprof format
	log.Printf("Converting perf.data to pprof format")
	pprofCmd := exec.Command("pprof", "-proto", "-output", pprofPath, perfDataPath)

	var pprofStderr bytes.Buffer
	pprofCmd.Stderr = &pprofStderr

	if err := pprofCmd.Run(); err != nil {
		log.Printf("pprof conversion failed: %v", err)
		log.Printf("pprof stderr: %s", pprofStderr.String())

		stderrStr := pprofStderr.String()
		if strings.Contains(stderrStr, "no samples") {
			http.Error(w, "No samples found in perf.data - process may have been idle during profiling", http.StatusBadRequest)
		} else if strings.Contains(stderrStr, "permission denied") {
			http.Error(w, "Permission denied accessing perf.data file", http.StatusForbidden)
		} else {
			http.Error(w, fmt.Sprintf("pprof conversion failed: %v\nStderr: %s", err, stderrStr), http.StatusInternalServerError)
		}
		return
	}

	// Check if pprof file was created and has content
	if stat, err := os.Stat(pprofPath); err != nil {
		http.Error(w, "pprof file was not created", http.StatusInternalServerError)
		return
	} else if stat.Size() == 0 {
		http.Error(w, "pprof file is empty - conversion produced no data", http.StatusInternalServerError)
		return
	}

	// Step 3: Serve the pprof file
	pprofFile, err := os.Open(pprofPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open pprof file: %v", err), http.StatusInternalServerError)
		return
	}
	defer pprofFile.Close()

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=profile-%s-%d.pb.gz", pid, duration))

	// Stream the file to the client
	if _, err := io.Copy(w, pprofFile); err != nil {
		log.Printf("Failed to stream pprof file: %v", err)
		return
	}

	log.Printf("Successfully served pprof profile for PID %s", pid)
}

// runBCCProfile executes the original BCC-based profiling for folded format
func runBCCProfile(w http.ResponseWriter, r *http.Request, pid string, duration int) {
	// Original BCC implementation for folded format
	args := []string{
		"profile-bpfcc",
		"-p", pid,
		"-F", "999",
		"-f",                        // folded format
		fmt.Sprintf("%d", duration), // duration as positional argument
	}

	cmd := exec.Command("sudo", args...)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("Running command: sudo %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		log.Printf("Command failed: %v", err)
		log.Printf("Stderr: %s", stderr.String())
		http.Error(w, fmt.Sprintf("Profiler failed: %v\nStderr: %s", err, stderr.String()), http.StatusInternalServerError)
		return
	}

	// Set headers for folded format
	w.Header().Set("Content-Type", "text/plain")

	// Return the output
	w.Write(stdout.Bytes())
}

func generateMockProfile(pid string, duration int) string {
	return fmt.Sprintf(`# Mock profile data for PID %s, duration %d seconds
main;runtime.main;main.main;net/http.ListenAndServe;net/http.(*Server).Serve 10
main;runtime.main;main.main;net/http.ListenAndServe;net/http.(*Server).Serve;net/http.(*conn).serve 25
main;runtime.main;main.main;net/http.ListenAndServe;net/http.(*Server).Serve;net/http.(*conn).serve;net/http.serverHandler.ServeHTTP 15
redis-server;main;aeMain;aeProcessEvents;aeApiPoll 50
redis-server;main;aeMain;aeProcessEvents;processCommand;lookupCommand 30
redis-server;main;aeMain;aeProcessEvents;processCommand;call 40
`, pid, duration)
}
