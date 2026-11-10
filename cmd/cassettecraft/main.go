package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/cassettecraft/internal/api"
	"github.com/alexandrmotologa/cassettecraft/internal/cassette"
	"github.com/alexandrmotologa/cassettecraft/internal/certs"
	"github.com/alexandrmotologa/cassettecraft/internal/config"
	"github.com/alexandrmotologa/cassettecraft/internal/proxy"
	"github.com/alexandrmotologa/cassettecraft/internal/runner"
	"github.com/alexandrmotologa/cassettecraft/web"
)

var (
	version = "1.0.0"
	commit  = "dev"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "start":
		runStart(os.Args[2:])
	case "record":
		runExecShortcut(config.ModeRecord, os.Args[2:])
	case "replay":
		runExecShortcut(config.ModeReplay, os.Args[2:])
	case "auto":
		runExecShortcut(config.ModeAuto, os.Args[2:])
	case "exec":
		runExec(os.Args[2:])
	case "studio":
		runStudio(os.Args[2:])
	case "ca":
		runCA(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("cassettecraft v%s (%s) %s/%s\n", version, commit, runtime.GOOS, runtime.GOARCH)
	case "help", "-h", "--help":
		printUsage()
	default:
		// If first arg looks like flags for start, run start
		if strings.HasPrefix(command, "-") {
			runStart(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'cassettecraft help' for usage.\n", command)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`cassettecraft v%s: Self-Hosted HTTP Recording & Replay Proxy

Usage:
  cassettecraft [command] [flags]

Core Commands:
  start               Start the proxy daemon with embedded React 19 Studio
  record   -- <cmd>   Run command in record mode with proxy environment pre-set
  replay   -- <cmd>   Run command in replay mode with zero internet calls
  auto     -- <cmd>   Run command in auto mode (replay existing, record new)
  exec     -- <cmd>   Run command with configurable proxy mode
  studio              Start proxy and open the Cassette Studio in default browser
  ca       [export]   Manage Root Certificate Authority (export, fingerprint)
  version             Print version and build details

Flags (start / studio):
  -p, --port          Proxy port (default: 8585)
  -m, --mode          Proxy mode: record, replay, auto, passthrough (default: record)
  -c, --cassette      Active cassette file name (default: default.yaml)
  -d, --cassette-dir  Directory for cassette storage (default: ./cassettes)
  --cert-dir          Directory for Root CA files (default: ~/.cassettecraft/ca)
  --latency           Inject synthetic latency (e.g. 50ms)
  --insecure          Skip upstream TLS verification (for local test upstreams)
  --no-studio         Disable embedded web studio

Examples:
  cassettecraft start -p 8585 -m record
  cassettecraft record -c stripe.yaml -- pytest tests/
  cassettecraft replay -c stripe.yaml -- go test -v ./...
  cassettecraft exec -m replay -d ./tests/cassettes -- npm test
  cassettecraft ca export --out ./ca.crt
`, version)
}

func runStart(args []string) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.Int("port", 8585, "Proxy port")
	fs.IntVar(port, "p", 8585, "Proxy port (shorthand)")
	modeStr := fs.String("mode", "record", "Proxy mode (record, replay, auto, passthrough)")
	fs.StringVar(modeStr, "m", "record", "Proxy mode (shorthand)")
	cassName := fs.String("cassette", "default.yaml", "Active cassette file name")
	fs.StringVar(cassName, "c", "default.yaml", "Active cassette (shorthand)")
	cassDir := fs.String("cassette-dir", "./cassettes", "Cassette directory")
	fs.StringVar(cassDir, "d", "./cassettes", "Cassette directory (shorthand)")
	certDir := fs.String("cert-dir", "", "Root CA storage directory")
	latencyStr := fs.String("latency", "", "Synthetic latency duration (e.g. 50ms)")
	insecure := fs.Bool("insecure", false, "Skip upstream TLS certificate verification")
	noStudio := fs.Bool("no-studio", false, "Disable web studio")
	configPath := fs.String("config", "", "Path to cassettecraft.yaml config")

	_ = fs.Parse(args)

	cfg, err := config.LoadConfigFromFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		cfg = config.DefaultConfig()
	}

	if *port != 8585 || cfg.Port == 0 {
		cfg.Port = *port
	}
	if *modeStr != "" {
		cfg.Mode = config.Mode(*modeStr)
	}
	if *cassName != "" {
		cfg.ActiveCass = *cassName
	}
	if *cassDir != "" {
		cfg.CassetteDir = *cassDir
	}
	if *certDir != "" {
		cfg.CertDir = *certDir
	}
	if *insecure {
		cfg.InsecureSkipVerify = true
	}
	if *noStudio {
		cfg.EnableStudio = false
	}
	if *latencyStr != "" {
		if dur, err := time.ParseDuration(*latencyStr); err == nil {
			cfg.Chaos.Latency = dur
		}
	}

	startDaemon(cfg, false)
}

func runStudio(args []string) {
	fs := flag.NewFlagSet("studio", flag.ExitOnError)
	port := fs.Int("port", 8585, "Proxy port")
	fs.IntVar(port, "p", 8585, "Proxy port (shorthand)")
	_ = fs.Parse(args)

	cfg := config.DefaultConfig()
	cfg.Port = *port
	cfg.EnableStudio = true

	startDaemon(cfg, true)
}

func startDaemon(cfg *config.Config, openBrowser bool) {
	// Initialize Root CA
	ca, err := certs.NewCA(cfg.CertDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing Root CA: %v\n", err)
		os.Exit(1)
	}

	// Initialize Storage
	storage, err := cassette.NewStorage(cfg.CassetteDir, cfg.ActiveCass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing cassette storage: %v\n", err)
		os.Exit(1)
	}

	// Initialize Proxy Server
	server := proxy.NewServer(cfg, ca, storage)

	// Attach Web Studio & REST API
	var webFS http.FileSystem
	if cfg.EnableStudio {
		if fs, err := web.GetFileSystem(); err == nil {
			webFS = fs
		}
	}
	apiHandler := api.NewHandler(server, ca, webFS)
	server.SetAPIHandler(apiHandler)

	fmt.Printf("\n  +------------------------------------------------------+\n")
	fmt.Printf("  |  CassetteCraft Proxy Daemon v%-10s              |\n", version)
	fmt.Printf("  +------------------------------------------------------+\n")
	fmt.Printf("  Proxy Address:     http://localhost:%d\n", cfg.Port)
	fmt.Printf("  Operating Mode:    %s\n", strings.ToUpper(string(cfg.Mode)))
	fmt.Printf("  Active Cassette:   %s\n", storage.GetActiveName())
	fmt.Printf("  Storage Directory: %s\n", cfg.CassetteDir)
	fmt.Printf("  Root CA Fingerprint: %s\n", ca.FingerprintSHA256()[:16]+"...")
	if cfg.EnableStudio {
		fmt.Printf("  Cassette Studio:   http://localhost:%d\n", cfg.Port)
	}
	fmt.Printf("\n  Press Ctrl+C to terminate the proxy.\n\n")

	if openBrowser && cfg.EnableStudio {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openURL(fmt.Sprintf("http://localhost:%d", cfg.Port))
		}()
	}

	// Trap shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nGracefully shutting down CassetteCraft proxy...")
		_ = server.Close()
		os.Exit(0)
	}()

	if err := server.Start(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Proxy failed to start: %v\n", err)
		os.Exit(1)
	}
}

func runExecShortcut(mode config.Mode, args []string) {
	newArgs := append([]string{"-m", string(mode)}, args...)
	runExec(newArgs)
}

func runExec(args []string) {
	// Split flags from the child command after "--"
	var flagArgs []string
	var cmdArgs []string

	splitIdx := -1
	for i, arg := range args {
		if arg == "--" {
			splitIdx = i
			break
		}
	}

	if splitIdx >= 0 {
		flagArgs = args[:splitIdx]
		if splitIdx+1 < len(args) {
			cmdArgs = args[splitIdx+1:]
		}
	} else {
		flagArgs = args
	}

	if len(cmdArgs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No command specified after '--'. Example:\n")
		fmt.Fprintf(os.Stderr, "  cassettecraft exec --mode record -- pytest tests/\n")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("exec", flag.ExitOnError)
	modeStr := fs.String("mode", "record", "Proxy mode: record, replay, auto, passthrough")
	fs.StringVar(modeStr, "m", "record", "Proxy mode (shorthand)")
	cassName := fs.String("cassette", "", "Target cassette file name")
	fs.StringVar(cassName, "c", "", "Target cassette (shorthand)")
	cassDir := fs.String("cassette-dir", "./cassettes", "Cassette directory")
	fs.StringVar(cassDir, "d", "./cassettes", "Cassette directory (shorthand)")
	certDir := fs.String("cert-dir", "", "Root CA storage directory")
	insecure := fs.Bool("insecure", false, "Skip upstream TLS certificate verification")

	_ = fs.Parse(flagArgs)

	cfg := config.DefaultConfig()
	cfg.Mode = config.Mode(*modeStr)
	if *cassName != "" {
		cfg.ActiveCass = *cassName
	}
	if *cassDir != "" {
		cfg.CassetteDir = *cassDir
	}
	if *certDir != "" {
		cfg.CertDir = *certDir
	}
	if *insecure {
		cfg.InsecureSkipVerify = true
	}
	cfg.EnableStudio = false

	// Initialize Root CA
	ca, err := certs.NewCA(cfg.CertDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing Root CA: %v\n", err)
		os.Exit(1)
	}

	// Initialize Storage
	storage, err := cassette.NewStorage(cfg.CassetteDir, cfg.ActiveCass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing cassette storage: %v\n", err)
		os.Exit(1)
	}

	// Bind proxy on an available local TCP port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bind ephemeral proxy port: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	proxyPort := listener.Addr().(*net.TCPAddr).Port
	cfg.Port = proxyPort

	server := proxy.NewServer(cfg, ca, storage)
	go func() {
		_ = server.StartWithListener(listener)
	}()
	defer server.Close()

	// Wait briefly for listener to be active
	time.Sleep(50 * time.Millisecond)

	// Determine actual CA directory path for cert exports
	actualCertDir := cfg.CertDir
	if actualCertDir == "" {
		home, _ := os.UserHomeDir()
		actualCertDir = filepath.Join(home, ".cassettecraft", "ca")
	}

	childCmd := cmdArgs[0]
	childCmdArgs := cmdArgs[1:]

	exitCode, err := runner.RunCommandWithProxy(runner.ExecOptions{
		ProxyPort: proxyPort,
		CertDir:   actualCertDir,
		Command:   childCmd,
		Args:      childCmdArgs,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Process execution error: %v\n", err)
	}

	os.Exit(exitCode)
}

func runCA(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: cassettecraft ca [export|fingerprint]")
		os.Exit(0)
	}

	sub := args[0]
	home, _ := os.UserHomeDir()
	certDir := filepath.Join(home, ".cassettecraft", "ca")

	ca, err := certs.NewCA(certDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading CA: %v\n", err)
		os.Exit(1)
	}

	switch sub {
	case "export":
		fs := flag.NewFlagSet("ca export", flag.ExitOnError)
		outPath := fs.String("out", "./ca.crt", "Destination path for ca.crt")
		_ = fs.Parse(args[1:])

		if err := ca.ExportCertToFile(*outPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to export certificate: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Root CA certificate exported to: %s\n", *outPath)
	case "fingerprint":
		fmt.Printf("SHA-256 Fingerprint: %s\n", ca.FingerprintSHA256())
	default:
		fmt.Fprintf(os.Stderr, "Unknown ca subcommand: %s\n", sub)
		os.Exit(1)
	}
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
