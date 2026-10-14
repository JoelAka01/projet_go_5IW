// Command ssh-server expose les TUI client et admin via SSH (bibliothèque wish).
//
//	ssh -p 2222 localhost   -> interface client
//	ssh -p 2223 localhost   -> interface admin
//
// Ce binaire ne contient aucune logique métier : il se contente d'ouvrir, pour
// chaque connexion SSH, un programme Bubble Tea branché sur l'API HTTP
// (cmd/server), exactement comme le font cmd/client et cmd/admin en local.
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

// modelFactory construit un modèle racine à partir de l'URL de l'API.
type modelFactory func(serverAddr string) tea.Model

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur:", err)
		os.Exit(1)
	}
}

func run() error {
	// Même mécanisme que cmd/client et cmd/admin : SERVER_ADDR, avec
	// http://localhost:8080 par défaut. Le flag --api permet de le surcharger.
	apiAddr := flag.String("api", envOr("SERVER_ADDR", "http://localhost:8080"), "URL de l'API HTTP")
	clientAddr := flag.String("client-addr", envOr("SSH_CLIENT_ADDR", ":2222"), "adresse d'écoute SSH de l'interface client")
	adminAddr := flag.String("admin-addr", envOr("SSH_ADMIN_ADDR", ":2223"), "adresse d'écoute SSH de l'interface admin")
	flag.Parse()

	// Deux ports plutôt qu'un seul serveur qui "devine" l'interface voulue :
	//  - chaque port correspond à une seule interface, sans routage fragile
	//    (nom d'utilisateur SSH, commande, menu de choix...) ;
	//  - le port admin peut être filtré séparément (pare-feu, VPN, bind sur
	//    127.0.0.1 via --admin-addr) sans toucher à l'accès client.
	// Les deux serveurs partagent la même clé d'hôte : c'est la même machine.
	clientSrv, err := newServer(*clientAddr, *apiAddr, client.NewModel)
	if err != nil {
		return fmt.Errorf("serveur SSH client: %w", err)
	}
	adminSrv, err := newServer(*adminAddr, *apiAddr, admin.NewModel)
	if err != nil {
		return fmt.Errorf("serveur SSH admin: %w", err)
	}

	// On ouvre les sockets de manière synchrone AVANT de lancer les goroutines :
	// un port déjà occupé est ainsi détecté immédiatement, avec un message
	// clair, au lieu d'être découvert plus tard dans une goroutine.
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
		// L'un des serveurs s'est arrêté de façon inattendue : on arrête
		// l'autre aussi plutôt que de tourner à moitié.
		log.Printf("arrêt suite à une erreur: %v", runErr)
	}

	// Shutdown ferme les listeners puis attend la fin des sessions en cours,
	// dans la limite de 30s (au-delà, les connexions restantes sont coupées).
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

// newServer crée un serveur wish qui lance, pour chaque session, un modèle
// produit par factory.
//
// Authentification : aucun handler d'auth SSH n'est configuré, donc toute
// connexion SSH est acceptée. C'est volontaire : l'authentification reste
// côté API HTTP (formulaire login/mot de passe des TUI, jeton vérifié par
// cmd/server, rôle admin contrôlé par l'API). SSH ne sert ici que de
// transport pour le terminal ; dupliquer la gestion des comptes en clés SSH
// créerait une seconde source de vérité à maintenir. Même via le port admin,
// un utilisateur sans rôle admin ne peut rien faire, car l'API le refuse.
func newServer(addr, apiAddr string, factory modelFactory) (*ssh.Server, error) {
	handler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		// Un NOUVEAU modèle par session : le modèle contient l'état de
		// l'utilisateur (jeton d'auth, panier, écran courant, formulaires).
		// Un modèle partagé mélangerait les sessions de plusieurs personnes
		// (fuite de jeton, données croisées) et provoquerait des accès
		// concurrents non protégés depuis plusieurs tea.Program.
		return factory(apiAddr), []tea.ProgramOption{tea.WithAltScreen()}
	}

	return wish.NewServer(
		wish.WithAddress(addr),
		// La clé d'hôte est générée au premier lancement si absente, puis
		// réutilisée : les clients SSH ne voient pas d'alerte "host key
		// changed" à chaque redémarrage. Le dossier .ssh/ est dans .gitignore.
		wish.WithHostKeyPath(hostKeyPath),
		// Les middlewares s'exécutent du dernier au premier : logging, puis
		// activeterm, puis Bubble Tea.
		wish.WithMiddleware(
			// Les styles de ui/style sont des variables globales lipgloss : leur
			// profil de couleurs serait sinon déduit de la sortie du serveur
			// (souvent sans TTY, donc sans couleur). On force donc la palette
			// 256 couleurs utilisée par les styles du projet.
			bubbletea.MiddlewareWithColorProfile(handler, termenv.ANSI256),
			// Refuse les sessions sans PTY (ex: `ssh host commande`, scp) :
			// une TUI a besoin d'un vrai terminal (taille, mode raw, touches),
			// sinon l'affichage est illisible et le programme ne reçoit pas
			// correctement les entrées.
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
}

// serve fait tourner srv sur ln et remonte toute erreur autre qu'un arrêt
// volontaire (Shutdown).
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
