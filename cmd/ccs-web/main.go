// Command ccs-web serves the Claude Code sessions dashboard on localhost.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"claude-sessions/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7777", "address to bind (localhost only)")
	flag.Parse()

	if err := run(*addr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(addr string) error {
	server, err := web.NewServer()
	if err != nil {
		return err
	}
	fmt.Printf("ccs-web listening on http://%s\n", addr)
	return http.ListenAndServe(addr, server)
}
