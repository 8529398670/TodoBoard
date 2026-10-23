// Command TodoBoard is the web server entry point.
//
// This file only wires things together -- load config, open the database,
// build the app, register routes, start and stop cleanly. Every decision of
// substance lives in a package under server/. If you find yourself adding
// business logic here, it belongs in server/routes/ or a new package instead.
//
// The same binary is also the admin CLI:
//
//	todoboard                     run the server
//	todoboard manage list-users   account administration
//	todoboard config check        create, check, or read config.yaml
//	todoboard health              the container's healthcheck
//	todoboard version             build information
package main

import (
	fmt "fmt"
	log "log"
	os "os"
	signal "os/signal"
	runtime "runtime"
	syscall "syscall"
	time "time"
	// The IANA timezone database, built in. The Docker image (Alpine, no
	// tzdata package) and a bare server may have none on disk, and then
	// every named timezone -- reminders, display settings, exports -- fails
	// to load. Costs about 450 KB.
	_ "time/tzdata"

	fiber "github.com/gofiber/fiber/v3"

	assets "todoboard/server/assets"
	bootstrap "todoboard/server/bootstrap"
	config "todoboard/server/config"
	control "todoboard/server/control"
	db "todoboard/server/db"
	healthcheck "todoboard/server/healthcheck"
	language "todoboard/server/language"
	manage "todoboard/server/manage"
	middleware "todoboard/server/middleware"
	models "todoboard/server/models"
	notify "todoboard/server/notify"
	routes "todoboard/server/routes"
	security "todoboard/server/security"
	static "todoboard/server/static"
)

// Stamped by run.sh, build.sh and the Docker build via -ldflags: the version
// is the build's date and time. The defaults are what you get from a plain
// `go build`, which is a useful signal in itself.
var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	// Subcommands before anything else: `manage` must work on a database the
	// server cannot currently open, which is exactly when you need it.
	if len( os.Args ) > 1 {
		switch os.Args[ 1 ] {
		case "manage":
			if err := manage.Run( os.Args[ 2: ] ); err != nil {
				fmt.Fprintf( os.Stderr , "error: %v\n" , err )
				os.Exit( 1 )
			}
			return
		case "config":
			if err := config.RunCommand( os.Args[ 2: ] ); err != nil {
				fmt.Fprintf( os.Stderr , "error: %v\n" , err )
				os.Exit( 1 )
			}
			return
		case "health":
			if err := healthcheck.Run(); err != nil {
				fmt.Fprintf( os.Stderr , "unhealthy: %v\n" , err )
				os.Exit( 1 )
			}
			return
		case "version" , "-version" , "--version":
			fmt.Printf( "TodoBoard %s (built %s, %s/%s, %s)\n" ,
				version , buildDate , runtime.GOOS , runtime.GOARCH , runtime.Version() )
			return
		case "-h" , "--help" , "help":
			fmt.Printf( "TodoBoard %s\n\nUsage:\n  todoboard                 run the web server\n  todoboard manage <cmd>    account administration\n  todoboard config <cmd>    create, check, or read config.yaml\n  todoboard health          exit 0 if the local server answers\n  todoboard version         build information\n\n" , version )
			manage.Usage()
			fmt.Println( "" )
			config.CommandUsage()
			return
		}
	}

	if err := run(); err != nil {
		log.Fatalf( "[fatal] %v" , err )
	}
}

func run() ( err error ) {
	cfg , err := config.Load()
	if err != nil { return }

	embedded , err := embeddedStatic()
	if err != nil { return }

	// Disk when it exists, the embedded copy otherwise. See server/assets.
	bundle , err := assets.Resolve( cfg.StaticDir , cfg.LanguageFile , embedded , embeddedLanguage )
	if err != nil { return }

	store , err := db.Open( cfg )
	if err != nil { return }
	defer store.Close()

	lang , err := language.Parse( bundle.LanguageYAML )
	if err != nil { return }

	static_server , err := static.New( bundle.Static )
	if err != nil { return }

	if err = bootstrap.EnsureFirstAdmin( store , cfg , "Admin" ); err != nil { return }
	if err = models.EnsureBoardSlugs( store ); err != nil { return }

	// The control socket is what lets `manage` run against a live server.
	// bolt hands the database to exactly one process, so without it every
	// admin command would need the server stopped first. It is started after
	// the database opens, which means the socket existing is itself proof
	// that a live process holds the lock.
	control_server , err := control.Listen( store , cfg )
	if err != nil { return }
	defer control_server.Close()

	guard := security.New( store , cfg )

	// Push notifications: off (and every publish a no-op) when
	// pushover.app_token is empty.
	notifier := notify.New( store , cfg , lang )
	notifier.Start()
	defer notifier.Stop()

	handlers := routes.New( store , cfg , guard , lang , static_server , notifier )

	app := fiber.New( fiber.Config{
		AppName: cfg.ProjectName,
		// An empty Server header keeps the response from advertising which
		// framework and version is running -- free reconnaissance otherwise.
		ServerHeader: "",
		BodyLimit:    cfg.BodyLimit,
		// Timeouts are what stop a handful of slow or stalled connections
		// from holding sockets open until the process runs out of them.
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  75 * time.Second,
		// Forwarded headers are believed only from the configured proxies,
		// and only once validated as IP addresses -- see server.trust_proxy
		// in config.yaml. Without the allowlist fiber trusts no proxy at all.
		TrustProxy:         cfg.TrustProxy,
		TrustProxyConfig:   fiber.TrustProxyConfig{ Proxies: cfg.TrustedProxies },
		EnableIPValidation: true,
		ProxyHeader:        proxyHeader( cfg ),
		ErrorHandler: func( c fiber.Ctx , handler_err error ) ( response_err error ) {
			// Never hand an internal error message to a client. The detail
			// goes to the log, where the operator can see it.
			code := fiber.StatusInternalServerError
			var fiber_err *fiber.Error
			if ok := asFiberError( handler_err , &fiber_err ); ok {
				code = fiber_err.Code
			}
			if code >= 500 {
				log.Printf( "[error] %s %s: %v" , c.Method() , c.Path() , handler_err )
			}
			response_err = c.Status( code ).JSON( fiber.Map{ "error": statusText( code ) } )
			return
		},
	} )

	app.Use( middleware.Recover( cfg ) )
	app.Use( middleware.SecurityHeaders( cfg ) )
	app.Use( middleware.RateLimit( cfg ) )

	handlers.Register( app )

	stop_purging := startPurgeLoop( store )
	defer close( stop_purging )

	// Shut down on a signal rather than being killed, so in-flight requests
	// finish and bolt closes cleanly instead of leaving a locked file behind.
	shutdown := make( chan os.Signal , 1 )
	signal.Notify( shutdown , os.Interrupt , syscall.SIGTERM )
	go func() {
		<-shutdown
		log.Println( "[shutdown] draining connections..." )
		// Close the control socket first: it is the door other processes
		// come through, and letting a manage command start against a
		// database that is about to close would fail confusingly.
		control_server.Close()
		app.ShutdownWithTimeout( 10 * time.Second )
	}()

	log.Printf( "[start] %s %s listening on %s (secure cookies: %v)" , cfg.ProjectName , version , cfg.ListenAddress() , cfg.SecureCookies )
	if cfg.ConfigCreated {
		log.Printf( "[config] %s (created, with a new secret key -- back it up)" , cfg.ConfigFile )
	} else {
		log.Printf( "[config] %s" , cfg.ConfigFile )
	}
	if cfg.ConfigWarning != "" {
		log.Printf( "[config] warning: %s" , cfg.ConfigWarning )
	}
	log.Printf( "[state]  %s" , cfg.AppDir )
	log.Printf( "[assets] %s" , bundle.Source )
	log.Printf( "[control] %s" , control_server.Path() )
	err = app.Listen( cfg.ListenAddress() , fiber.ListenConfig{ DisableStartupMessage: true } )
	return
}

func proxyHeader( cfg *config.Config ) ( result string ) {
	if cfg.TrustProxy {
		result = cfg.ProxyHeader
	}
	return
}

func asFiberError( err error , out **fiber.Error ) ( ok bool ) {
	converted , ok := err.( *fiber.Error )
	if ok {
		*out = converted
	}
	return
}

func statusText( code int ) ( result string ) {
	switch {
	case code == fiber.StatusNotFound:
		result = "not found"
	case code == fiber.StatusUnauthorized:
		result = "not authenticated"
	case code == fiber.StatusForbidden:
		result = "forbidden"
	case code == fiber.StatusTooManyRequests:
		result = "too many requests"
	case code >= 500:
		result = "internal error"
	default:
		result = "bad request"
	}
	return
}

// startPurgeLoop drops expired sessions, expired login tokens, and the API
// keys and invite links that have been dead long enough to stop being worth
// listing. None of it is a correctness issue -- every one of those is
// already refused on read -- so this runs quietly in the background purely to
// keep the bolt file from growing forever.
func startPurgeLoop( store *db.Store ) ( stop chan struct{} ) {
	stop = make( chan struct{} )
	go func() {
		ticker := time.NewTicker( time.Hour )
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				sessions , _ := models.PurgeExpiredSessions( store )
				tokens , _ := models.PurgeExpiredLoginTokens( store )
				keys , _ := models.PurgeDeadAPIKeys( store )
				invites , _ := models.PurgeDeadInviteLinks( store )
				if sessions > 0 || tokens > 0 || keys > 0 || invites > 0 {
					fmt.Printf( "[purge] removed %d expired sessions, %d expired login tokens, %d dead api keys, %d dead invite links\n" , sessions , tokens , keys , invites )
				}
			}
		}
	}()
	return
}
