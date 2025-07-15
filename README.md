# bcc-exporter

**bcc-exporter** is a minimal HTTP server that wraps BCC tools like `profile-bpfcc` to expose Linux CPU profiling data over HTTP. It provides endpoints compatible with `pprof` and flamegraph tools, making it easy to collect and visualize profiling data from running processes.

## ✨ Features

- HTTP endpoints to trigger BCC profiling
- Folded stack output for use with Flamegraph
- Compatible with `go tool pprof` workflows
- Lightweight, no dependencies beyond Go and BCC

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

Returns folded stack traces with a Content-Type: application/octet-stream header (mimicking pprof endpoint structure). Can be adapted to binary pprof format later.

**Example:**
```bash
curl -o profile.txt "http://localhost:8080/debug/pprof/profile?pid=`pgrep redis`&seconds=10"
```

**Test Mode:**
```bash
curl -o profile.txt "http://localhost:8080/debug/pprof/profile?pid=`pgrep redis`&seconds=10&test=true"
```

## 🔧 Requirements

- Linux with BPF support (kernel 4.9+ recommended)
- bpfcc-tools installed (profile-bpfcc must be available)
- sudo access or appropriate capabilities to run BCC tools

**Install dependencies:**
```bash
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

- Convert folded format to binary .pb.gz for go tool pprof compatibility
- Add wrappers for additional BCC tools (e.g., offcputime-bpfcc, tcplife-bpfcc)
- Add Prometheus-compatible metrics endpoints
- Dockerfile and systemd service support

## 🔍 Troubleshooting

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