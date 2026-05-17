package repscli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Prasad-178/reps/internal/server"
	"github.com/urfave/cli/v3"
)

func ServeCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "start HTTP API for the web UI",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "addr", Usage: "bind address", Value: ":7777"},
			&cli.StringFlag{Name: "origins", Usage: "comma-separated CORS allowlist",
				Value: "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001"},
		},
		Action: serveAction,
	}
}

func serveAction(ctx context.Context, c *cli.Command) error {
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	if cfg.LLM.APIKey == "" {
		fmt.Println("WARNING: OPENROUTER_API_KEY not set. /api/drill/* will fail; read-only endpoints work.")
	}
	srv := server.New(*cfg, s, client)
	if v := c.String("origins"); v != "" {
		srv.Origins = splitTrim(v, ",")
	}

	addr := c.String("addr")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		// no write timeout — SSE drill streams can be long
	}
	log.Printf("reps serve → %s (CORS: %v)", addr, srv.Origins)
	return httpSrv.ListenAndServe()
}

func splitTrim(s, sep string) []string {
	raw := strings.Split(s, sep)
	out := raw[:0]
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r != "" {
			out = append(out, r)
		}
	}
	return out
}
