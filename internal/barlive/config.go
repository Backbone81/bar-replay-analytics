package barlive

import (
	"net/http"

	"golang.org/x/time/rate"
)

// Config configures a Client. NewClient starts from DefaultConfig and applies its options to it.
type Config struct {
	// Transport executes the HTTP requests. Defaults to http.DefaultTransport.
	Transport http.RoundTripper

	// TeiserverURL is the base URL of Teiserver.
	TeiserverURL string

	// ReplayAPIURL is the base URL of bar-db.
	ReplayAPIURL string

	// StorageURL is the base URL of the replay storage.
	StorageURL string

	// MaxRequestRate is the maximum number of requests per second to Teiserver and bar-db together, shared by all
	// goroutines using the Client. Downloads from the storage do not count. Zero means no limit. Defaults to 1.
	MaxRequestRate rate.Limit
}

// DefaultConfig is the configuration of the real BAR Live Services.
var DefaultConfig = Config{
	Transport:      http.DefaultTransport,
	TeiserverURL:   "https://server4.beyondallreason.info",
	ReplayAPIURL:   "https://api.bar-rts.com",
	StorageURL:     "https://storage.uk.cloud.ovh.net/v1/AUTH_10286efc0d334efd917d476d7183232e/BAR",
	MaxRequestRate: 1,
}
