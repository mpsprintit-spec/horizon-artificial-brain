// horizon-chat provides an offline line-oriented chat UI for Termux/terminals.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/project-horizon/horizon-core/services/ai/terminal"
)

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func run() (err error) {
	state := flag.String("state", "", "direktori profil chat (default: ~/.local/share/horizon/chat-v1)")
	noColor := flag.Bool("no-color", false, "nonaktifkan warna ANSI")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("gunakan -help untuk pilihan; ketik pesan setelah UI terbuka")
	}
	if *state == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		*state = filepath.Join(homeDir, ".local", "share", "horizon", "chat-v1")
	}
	fmt.Fprintln(os.Stderr, "Memuat Brain Horizon...")
	session, err := terminal.Open(*state)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, session.Close()) }()
	width, _ := strconv.Atoi(os.Getenv("COLUMNS"))
	_, noColorEnv := os.LookupEnv("NO_COLOR")
	return terminal.Run(session, os.Stdin, os.Stdout, terminal.Options{Color: !*noColor && !noColorEnv && os.Getenv("TERM") != "dumb" && isTerminal(os.Stdout), EchoInput: !isTerminal(os.Stdin), Width: width})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Horizon:", err)
		os.Exit(1)
	}
}
