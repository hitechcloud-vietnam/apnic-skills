// Package apnic provides a Go SDK for APNIC public data services.
//
// This is the root package of github.com/hitechcloud-vietnam/apnic-skills. The actual
// implementation lives in subpackages under internal/; this package wraps the
// transport.Client so that importers can keep using
// "github.com/hitechcloud-vietnam/apnic-skills" as the import path and apnic.Client /
// apnic.NewClient / apnic.WithChunkSize / etc. as before.
package apnic
