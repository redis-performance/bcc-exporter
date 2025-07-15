package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	http.HandleFunc("/debug/pprof/profile", handlePprof)
	http.HandleFunc("/debug/folded/profile", handleFolded)

	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
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

	// Real profiling - simplified arguments
	args := []string{
		"profile-bpfcc",
		"-p", pid,
		"-f",                   // folded format
		fmt.Sprintf("%d", dur), // duration as positional argument
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

	// Set headers
	if format == "pprof" {
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("Content-Type", "text/plain")
	}

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
