//go:build linux

package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"text/tabwriter"

	"wg0proxy/proxy"

	"golang.org/x/term"
)

var stdinReader = bufio.NewReader(os.Stdin)

func prompt(label string) string {
	fmt.Print(label)
	line, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptRequired(label string) string {
	for {
		v := prompt(label)
		if v != "" {
			return v
		}
	}
}

func promptPassword(label string) string {
	fmt.Print(label)
	if term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
		return strings.TrimSpace(string(b))
	}
	// Non-interactive: read from stdin
	line, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptPasswordRequired(label string) string {
	for {
		v := promptPassword(label)
		if v != "" {
			return v
		}
	}
}

func cmdAdminSet(store *proxy.Store) {
	username := promptRequired("Admin username: ")
	password := promptPasswordRequired("Admin password: ")

	if err := store.SetAdmin(username, password); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Admin credentials set for %q.\n", username)
}

func cmdUserAdd(store *proxy.Store) {
	username := promptRequired("Username: ")
	password := promptPasswordRequired("Password: ")

	if err := store.AddUser(username, password); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("User %q added successfully.\n", username)
}

func cmdUserDelete(store *proxy.Store) {
	username := promptRequired("Username to delete: ")
	confirm := prompt(fmt.Sprintf("Are you sure you want to delete %q? (y/n): ", username))
	if !strings.HasPrefix(strings.ToLower(confirm), "y") {
		fmt.Println("Cancelled.")
		return
	}

	if err := store.DeleteUser(username); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("User %q deleted.\n", username)
}

func cmdUserList(store *proxy.Store) {
	users, err := store.ListUsers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(users) == 0 {
		fmt.Println("No users found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USERNAME\tSTATUS\tUPLOAD\tDOWNLOAD\tCREATED")
	for _, u := range users {
		status := "enabled"
		if !u.Enabled {
			status = "disabled"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			u.Username, status, proxy.FormatBytes(u.Upload), proxy.FormatBytes(u.Download),
			u.CreatedAt.Format("2006-01-02 15:04"))
	}
	w.Flush()
}

func cmdUserURL(store *proxy.Store, cfg *proxy.Config) {
	username := promptRequired("Username: ")
	serverAddr := promptRequired("Server address (IP or domain): ")

	u, err := store.GetUser(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// SOCKS5 URL format for v2rayNG / v2box:
	// socks://base64(username:password)@host:port#remark
	creds := base64.URLEncoding.EncodeToString([]byte(username + ":" + u.Password))
	url := fmt.Sprintf("socks://%s@%s:%s#%s", creds, serverAddr, cfg.ListenPort, username)

	fmt.Println(url)
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: wg0proxy <command> [args]

Commands:
  serve                              Start the SOCKS5 proxy server
  admin set <username> <password>    Set admin credentials
  user add <username> <password>     Add a new user
  user delete <username>             Delete a user
  user list                          List all users with traffic stats
  user url <username> <server-addr>  Generate subscription URL for v2rayNG/v2box
`)
	os.Exit(1)
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := proxy.LoadConfig()
	dbPath := cfg.DataDir + "/wg0proxy.db"

	if len(os.Args) < 2 {
		usage()
	}

	store, err := proxy.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	switch os.Args[1] {
	case "serve":
		proxy.RunServe(&cfg, store)

	case "admin":
		if len(os.Args) < 3 {
			usage()
		}
		switch os.Args[2] {
		case "set":
			cmdAdminSet(store)
		default:
			usage()
		}

	case "user":
		if len(os.Args) < 3 {
			usage()
		}
		switch os.Args[2] {
		case "add":
			cmdUserAdd(store)
		case "delete":
			cmdUserDelete(store)
		case "list":
			cmdUserList(store)
		case "url":
			cmdUserURL(store, &cfg)
		default:
			usage()
		}

	default:
		usage()
	}
}
