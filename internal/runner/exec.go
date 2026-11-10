package runner

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

// ExecOptions configures the child process environment.
type ExecOptions struct {
	ProxyPort int
	CertDir   string
	Command   string
	Args      []string
}

// RunCommandWithProxy executes a command with all proxy and CA environment variables set.
func RunCommandWithProxy(opts ExecOptions) (int, error) {
	if opts.Command == "" {
		return 1, fmt.Errorf("no command provided to execute")
	}

	cmd := exec.Command(opts.Command, opts.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", opts.ProxyPort)
	caCertPath := filepath.Join(opts.CertDir, "ca.crt")

	// Inherit existing environment and inject proxy variables
	env := os.Environ()
	proxyEnvVars := []string{
		fmt.Sprintf("HTTP_PROXY=%s", proxyURL),
		fmt.Sprintf("http_proxy=%s", proxyURL),
		fmt.Sprintf("HTTPS_PROXY=%s", proxyURL),
		fmt.Sprintf("https_proxy=%s", proxyURL),
		fmt.Sprintf("ALL_PROXY=%s", proxyURL),
		fmt.Sprintf("all_proxy=%s", proxyURL),
		fmt.Sprintf("SSL_CERT_FILE=%s", caCertPath),
		fmt.Sprintf("NODE_EXTRA_CA_CERTS=%s", caCertPath),
		fmt.Sprintf("REQUESTS_CA_BUNDLE=%s", caCertPath),
		fmt.Sprintf("CURL_CA_BUNDLE=%s", caCertPath),
	}
	cmd.Env = append(env, proxyEnvVars...)

	// Handle interrupt signals and forward to child process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start command %q: %w", opts.Command, err)
	}

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}

	return 0, nil
}
