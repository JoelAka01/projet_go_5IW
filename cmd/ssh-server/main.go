package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"

	"ecommerce-cli/ui/admin"
	"ecommerce-cli/ui/client"
)

const (
	hostKeyPath     = ".ssh/id_ed25519"
	shutdownTimeout = 30 * time.Second
)

type modelFactory func(serverAddr string) tea.Model

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur:", err)
		os.Exit(1)
	}
}

func run() error {

	apiAddr := flag.String("api", envOr("SERVER_ADDR", "http://localhost:8080"), "URL de l'API HTTP")
	clientAddr := flag.String("client-addr", envOr("SSH_CLIENT_ADDR", ":2222"), "adresse d'écoute SSH de l'interface client")
	adminAddr := flag.String("admin-addr", envOr("SSH_ADMIN_ADDR", ":2223"), "adresse d'écoute SSH de l'interface admin")
	flag.Parse()

	clientSrv, err := newServer(*clientAddr, *apiAddr, client.NewModel)
	if err != nil {
		return fmt.Errorf("serveur SSH client: %w", err)
	}
	adminSrv, err := newServer(*adminAddr, *apiAddr, admin.NewModel)
	if err != nil {
		return fmt.Errorf("serveur SSH admin: %w", err)
	}

	clientLn, err := net.Listen("tcp", *clientAddr)
	if err != nil {
		return fmt.Errorf("écoute SSH client sur %s: %w", *clientAddr, err)
	}
	adminLn, err := net.Listen("tcp", *adminAddr)
	if err != nil {
		clientLn.Close()
		return fmt.Errorf("écoute SSH admin sur %s: %w", *adminAddr, err)
	}

	errCh := make(chan error, 2)
	go serve("client", clientSrv, clientLn, errCh)
	go serve("admin", adminSrv, adminLn, errCh)

	log.Printf("SSH client sur %s, SSH admin sur %s (API: %s)", *clientAddr, *adminAddr, *apiAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var runErr error
	select {
	case sig := <-sigCh:
		log.Printf("signal %s reçu, arrêt en cours...", sig)
	case runErr = <-errCh:

		log.Printf("arrêt suite à une erreur: %v", runErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var shutdownErrs []error
	for name, srv := range map[string]*ssh.Server{"client": clientSrv, "admin": adminSrv} {
		if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("arrêt du serveur %s: %w", name, err))
		}
	}

	if runErr != nil {
		return runErr
	}
	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}
	log.Println("serveurs SSH arrêtés proprement")
	return nil
}

func newServer(addr, apiAddr string, factory modelFactory) (*ssh.Server, error) {
	handler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {

		return factory(apiAddr), []tea.ProgramOption{tea.WithAltScreen()}
	}

	return wish.NewServer(
		wish.WithAddress(addr),

		wish.WithHostKeyPath(hostKeyPath),

		wish.WithMiddleware(

			bubbletea.MiddlewareWithColorProfile(handler, termenv.ANSI256),

			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
}

func serve(name string, srv *ssh.Server, ln net.Listener, errCh chan<- error) {
	if err := srv.Serve(ln); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		errCh <- fmt.Errorf("serveur SSH %s: %w", name, err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
