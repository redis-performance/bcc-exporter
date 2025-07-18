# bcc-exporter

**bcc-exporter** is a minimal HTTP server that provides Linux CPU profiling data over HTTP. It supports both modern `perf` + `pprof` workflows for binary pprof output and traditional BCC tools for folded stack traces, making it easy to collect and visualize profiling data from running processes.

## ✨ Features

- HTTP endpoints to trigger CPU profiling
- **Binary pprof format** using `perf record` + `pprof` conversion (fully compatible with `go tool pprof`)
- **Folded stack output** using BCC tools for use with Flamegraph


## 🚀 Endpoints

### `/debug/folded/profile`

Returns folded stack traces in text format (suitable for Flamegraph).

**Example:**
```bash
curl "http://localhost:8080/debug/folded/profile?pid=`pgrep redis`&seconds=10"
```

**Test Mode (when BCC is not available):**
```bash
curl "http://localhost:8080/debug/folded/profile?pid=`pgrep redis`&seconds=10&test=true"
```

### `/debug/pprof/profile`

Returns **binary pprof data** (.pb.gz format) using `perf record` + `pprof` conversion. Fully compatible with `go tool pprof` and other pprof-based tools.

**Example:**
```bash
# Download binary pprof file
curl -o profile.pb.gz "http://localhost:8080/debug/pprof/profile?pid=1234&seconds=30"

# Use directly with go tool pprof
go tool pprof http://localhost:8080/debug/pprof/profile?pid=1234&seconds=30
```

**Test Mode:**
```bash
curl -o profile.pb.gz "http://localhost:8080/debug/pprof/profile?pid=1234&seconds=10&test=true"
```

## 🔧 Requirements

### For pprof endpoint (binary format):
- Linux with perf support (kernel 3.7+ recommended)
- `perf` tools installed
- `pprof` tool installed
- Appropriate permissions for perf profiling

### For folded endpoint (text format):
- Linux with BPF support (kernel 4.9+ recommended)
- bpfcc-tools installed (profile-bpfcc must be available)
- sudo access or appropriate capabilities to run BCC tools

**Install dependencies:**
```bash
# For perf + pprof (recommended)
sudo apt-get install linux-perf
go install github.com/google/pprof@latest

# For BCC tools (folded format)
sudo apt-get install bpfcc-tools linux-headers-$(uname -r)
```

**Note:** If you encounter BCC library issues (like `undefined symbol` errors), you can use the test mode by adding `&test=true` to any request to see mock profiling data.

## 🛠️ Build and Run

```bash
go build -o bcc-exporter
sudo ./bcc-exporter
```

### Command Line Options

- `-port`: Specify the port to listen on (default: 8080)
- `-password`: Enable basic authentication with the specified password (optional)

**Examples:**

```bash
# Run on default port 8080
sudo ./bcc-exporter

# Run on custom port
sudo ./bcc-exporter -port 9090

# Run with basic authentication
sudo ./bcc-exporter -password mysecretpassword

# Run on custom port with authentication
sudo ./bcc-exporter -port 9090 -password mysecretpassword
```

When authentication is enabled, use username `admin` with your specified password:

```bash
# Example with authentication
curl -u admin:mysecretpassword "http://localhost:8080/debug/folded/profile?pid=`pgrep redis`&seconds=10"
```

## 📊 Using with go tool pprof

The `/debug/pprof/profile` endpoint generates binary pprof files that work seamlessly with `go tool pprof`:

```bash
# Interactive analysis
go tool pprof http://localhost:8080/debug/pprof/profile?pid=1234&seconds=30

# Generate web UI
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?pid=1234&seconds=30

# Generate flamegraph
go tool pprof -http=:8081 -flame http://localhost:8080/debug/pprof/profile?pid=1234&seconds=30

# Save profile for later analysis
curl -o myapp.pb.gz "http://localhost:8080/debug/pprof/profile?pid=1234&seconds=30"
go tool pprof myapp.pb.gz
```

## 🔥 Generate a Flamegraph

Use Brendan Gregg's Flamegraph tools to generate visual output from folded stack traces:

```bash
curl "http://localhost:8080/debug/folded/profile?pid=`pgrep redis`&seconds=10" > folded.txt

git clone https://github.com/brendangregg/Flamegraph
cd Flamegraph
./flamegraph.pl ../folded.txt > flame.svg
```

Open `flame.svg` in a browser to explore the flamegraph.

## 🧩 Extending

Planned or potential future extensions:

- Add wrappers for additional BCC tools (e.g., offcputime-bpfcc, tcplife-bpfcc)
- Add memory profiling support
- Add Prometheus-compatible metrics endpoints
- Dockerfile and systemd service support
- Support for custom perf events and sampling frequencies

## 🔍 Troubleshooting

### Perf Permission Issues

If you encounter "Permission denied" errors with the pprof endpoint:

```
perf record failed: Permission denied
```

This is usually due to perf security restrictions. Solutions:

1. **Run with sudo** (simplest):
   ```bash
   sudo ./bcc-exporter
   ```

2. **Adjust perf_event_paranoid** (system-wide):
   ```bash
   # Temporarily (until reboot)
   echo 1 | sudo tee /proc/sys/kernel/perf_event_paranoid

   # Permanently
   echo 'kernel.perf_event_paranoid = 1' | sudo tee -a /etc/sysctl.conf
   ```

3. **Add CAP_SYS_ADMIN capability**:
   ```bash
   sudo setcap cap_sys_admin+ep ./bcc-exporter
   ```

### Missing Tools

If you get "tool not found" errors:

```bash
# Install perf tools
sudo apt-get install linux-perf

# Install pprof
go install github.com/google/pprof@latest

# Verify installation
perf --version
pprof --help
```

### BCC Library Issues

If you encounter errors like:
```
OSError: /lib/x86_64-linux-gnu/libbcc.so.0: undefined symbol: _ZSt28__throw_bad_array_new_lengthv
```

This indicates a broken BCC installation. You can:

1. **Use test mode** to verify the server works:
   ```bash
   curl "http://localhost:8080/debug/folded/profile?pid=1234&seconds=5&test=true"
   ```

2. **Try reinstalling BCC** (Ubuntu/Debian):
   ```bash
   sudo apt-get remove --purge bpfcc-tools
   sudo apt-get install bpfcc-tools linux-headers-$(uname -r)
   ```

3. **Check BCC directly**:
   ```bash
   sudo profile-bpfcc -p $(pgrep redis) -f 2
   ```

### Permission Issues

The server needs sudo access to run BCC tools. Make sure:
- You run the server with `sudo ./bcc-exporter`
- Your user has sudo privileges
- BCC tools are installed and accessible