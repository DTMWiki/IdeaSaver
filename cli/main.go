package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		runSystemctl("start")
	case "stop":
		runSystemctl("stop")
	case "restart":
		runSystemctl("restart")
	case "status":
		runSystemctl("status")
	case "logs":
		runLogs()
	case "config":
		handleConfig()
	case "db":
		handleDB()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ideactl - IdeaSaver Management Tool

Usage:
  ideactl <command> [options]

Commands:
  start              Start the IdeaSaver service
  stop               Stop the IdeaSaver service
  restart            Restart the IdeaSaver service
  status             Show service status
  logs [--lines=N]   View service logs (default: follow mode)
  config list        List all configuration values
  config get <KEY>   Get a configuration value
  config set <K> <V> Set a configuration value
  db migrate         Run database migrations
  db status          Show migration status
  help               Show this help message`)
}

func runSystemctl(action string) {
	cmd := exec.Command("systemctl", action, "ideasaver")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to %s service: %v\n", action, err)
		os.Exit(1)
	}
	fmt.Printf("Service %sed successfully\n", action)
}

func runLogs() {
	args := []string{"-u", "ideasaver"}

	// Check for --lines flag
	linesMode := true
	for _, arg := range os.Args[2:] {
		if strings.HasPrefix(arg, "--lines=") {
			n := strings.TrimPrefix(arg, "--lines=")
			args = append(args, "-n", n)
			linesMode = false
		}
	}

	if linesMode {
		args = append(args, "-f") // follow mode by default
	}

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

const envFile = "/etc/ideasaver/.env"
const binaryFile = "/data/ideasaver/ideasaver"

func handleConfig() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: ideactl config <list|get|set>")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "list":
		data, err := os.ReadFile(envFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))

	case "get":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ideactl config get <KEY>")
			os.Exit(1)
		}
		key := os.Args[3]
		data, err := os.ReadFile(envFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
			os.Exit(1)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, key+"=") {
				fmt.Println(line)
				return
			}
		}
		fmt.Fprintf(os.Stderr, "Key not found: %s\n", key)
		os.Exit(1)

	case "set":
		if len(os.Args) < 5 {
			fmt.Println("Usage: ideactl config set <KEY> <VALUE>")
			os.Exit(1)
		}
		key := os.Args[3]
		value := os.Args[4]

		data, err := os.ReadFile(envFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
			os.Exit(1)
		}

		lines := strings.Split(string(data), "\n")
		found := false
		for i, line := range lines {
			if strings.HasPrefix(line, key+"=") {
				lines[i] = key + "=" + value
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, key+"="+value)
		}

		if err := os.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s=%s\n", key, value)
		fmt.Println("Run 'ideactl restart' to apply changes")

	default:
		fmt.Println("Usage: ideactl config <list|get|set>")
		os.Exit(1)
	}
}

func handleDB() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: ideactl db <migrate|status>")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "migrate":
		cmd := exec.Command(binaryFile, "migrate")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}

	case "status":
		fmt.Println("Checking migration status...")
		// Simple check: try to connect and verify tables exist
		cmd := exec.Command(binaryFile, "migrate")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()

	default:
		fmt.Println("Usage: ideactl db <migrate|status>")
		os.Exit(1)
	}
}
