package main

type Config struct {
	// The service address.
	Address string `split_words:"true" default:":9090"`

	// The authentication server address.
	ForwardAuthAddress string `split_words:"true" required:"true"`
	// A list of the headers to copy from the request to the authentication server.
	AuthRequestHeaders []string `split_words:"true"`
	// A list of the headers to copy from the authentication server to the request.
	AuthResponseHeaders []string `split_words:"true"`
	// Trust all the existing X-Forwarded-* headers.
	// This is needed to work with Pomerium.
	TrustForwardHeader bool `split_words:"true" default:"true"`

	// Options: debug, info, warn, error, dpanic, panic, and fatal.
	LogLevel string `split_words:"true" default:"info"`

	// Set during build / compile time.
	Version string `split_words:"true" default:"unknown"`
}
