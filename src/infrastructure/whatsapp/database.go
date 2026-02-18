package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// InitWaDB initializes the WhatsApp database connection
func InitWaDB(ctx context.Context, DBURI string) *sqlstore.Container {
	log = waLog.Stdout("Main", config.WhatsappLogLevel, true)
	dbLog := waLog.Stdout("Database", config.WhatsappLogLevel, true)

	storeContainer, err := initDatabase(ctx, dbLog, DBURI)
	if err != nil {
		log.Errorf("Database initialization error: %v", err)
		panic(pkgError.InternalServerError(fmt.Sprintf("Database initialization error: %v", err)))
	}

	return storeContainer
}

// initDatabase creates and returns a database store container based on the configured URI
func initDatabase(ctx context.Context, dbLog waLog.Logger, DBURI string) (*sqlstore.Container, error) {
	// Normalize commonly used SQLite DSN flags so they work with pure‑Go drivers
	// (e.g. modernc.org/sqlite expects PRAGMA via `_pragma=`).
	if strings.HasPrefix(DBURI, "file:") {
		// Accept both legacy `_foreign_keys=on|1` and convert to `_pragma=foreign_keys(1)`
		if strings.Contains(DBURI, "_foreign_keys=on") || strings.Contains(DBURI, "_foreign_keys=1") {
			DBURI = strings.ReplaceAll(DBURI, "_foreign_keys=on", "_pragma=foreign_keys(1)")
			DBURI = strings.ReplaceAll(DBURI, "_foreign_keys=1", "_pragma=foreign_keys(1)")
		}

		// Ensure WAL journal mode for better concurrency and enable shared cache
		// to reduce SQLITE_BUSY occurrences when multiple connections perform writes.
		if !strings.Contains(DBURI, "_journal_mode=") {
			if strings.Contains(DBURI, "?") {
				DBURI += "&_journal_mode=WAL"
			} else {
				DBURI += "?_journal_mode=WAL"
			}
		}
		if !strings.Contains(DBURI, "_cache=") {
			DBURI += "&_cache=shared"
		}

		// Add a busy timeout PRAGMA to reduce transient SQLITE_BUSY errors
		if !strings.Contains(DBURI, "busy_timeout") {
			DBURI += "&_pragma=busy_timeout(5000)"
		}

		return sqlstore.New(ctx, "sqlite", DBURI, dbLog)
	} else if strings.HasPrefix(DBURI, "postgres:") {
		return sqlstore.New(ctx, "postgres", DBURI, dbLog)
	}

	return nil, fmt.Errorf("unknown database type: %s. Currently only sqlite (file:) and postgres are supported", DBURI)
}

// GetConnectionStatus returns the current connection status of the global client
func GetConnectionStatus() (isConnected bool, isLoggedIn bool, deviceID string) {
	globalStateMu.RLock()
	currentClient := cli
	globalStateMu.RUnlock()
	if currentClient == nil {
		return false, false, ""
	}

	isConnected = currentClient.IsConnected()
	isLoggedIn = currentClient.IsLoggedIn()

	if currentClient.Store != nil && currentClient.Store.ID != nil {
		deviceID = currentClient.Store.ID.String()
	}

	return isConnected, isLoggedIn, deviceID
}
