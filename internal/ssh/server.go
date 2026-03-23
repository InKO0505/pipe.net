// internal/ssh/server.go
package ssh

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clinet/internal/db"
	"clinet/internal/pubsub"
	"clinet/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"
)

func Start(database *db.DB, broker *pubsub.Broker, port int) {
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf(":%d", port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		// Intercept the public key and pass it into the context
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			ctx.SetValue("pubkey", key)
			return true
		}),
		wish.WithMiddleware(
			bm.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
				pubkey := s.Context().Value("pubkey").(ssh.PublicKey)
				keyStr := string(gossh.MarshalAuthorizedKey(pubkey))

				// If User missing inject them in, otherwise look them up.
				user, err := database.GetUserByPubKey(keyStr)
				if err != nil {
					user = database.CreateUser(keyStr)
					log.Printf("New User created for public key: %s...\n", keyStr[:30])
				}

				m := tui.NewModel(database, broker, user, s)
				return m, []tea.ProgramOption{tea.WithAltScreen()}
			}),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalf("Could not start SSH Server: %s", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Starting SSH server on port :%d", port)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatalf("Server error: %s", err)
		}
	}()

	<-done
	log.Println("Stopping SSH server peacefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown failure: %s", err)
	}
}
