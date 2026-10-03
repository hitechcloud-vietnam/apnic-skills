package query

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// FetchRRDPNotification fetches and parses the RRDP notification.xml from the
// configured RRDP base URL. The notification identifies the current snapshot
// and the recent deltas for incremental synchronisation.
func FetchRRDPNotification(ctx context.Context, c *transport.Client) (*models.RRDPNotification, error) {
	url := transport.BuildRRDPNotificationURL(c.RRDPBaseURL())
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseRRDPNotification(strings.NewReader(body))
}

// FetchRRDPSnapshot fetches and parses an RRDP snapshot.xml from the given URI
// (taken from an RRDPNotification.Snapshot.URI). The snapshot is streamed: only
// the rsync URIs of <publish>/<withdraw> elements are retained, so memory use
// stays bounded even for the multi-megabyte snapshot files. A gzip
// Content-Encoding (the server applies it when the client advertises
// Accept-Encoding: gzip) is transparently decompressed.
func FetchRRDPSnapshot(ctx context.Context, c *transport.Client, uri string) (*models.RPKISnapshot, error) {
	resp, err := c.DoHTTPRequest(ctx, http.MethodGet, uri, "application/xml, text/xml")
	if err != nil {
		return nil, fmt.Errorf("RRDP snapshot request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d for URL: %s", resp.StatusCode, uri)
	}
	body := io.Reader(resp.Body)
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("RRDP snapshot gzip init failed: %w", err)
		}
		defer gz.Close()
		body = gz
	}
	return parseRPKISnapshot(body)
}

// FetchRRDPDelta fetches and parses an RRDP delta.xml from the given URI. Deltas
// share the snapshot format (<publish>/<withdraw> elements) but represent an
// incremental update at a specific serial.
func FetchRRDPDelta(ctx context.Context, c *transport.Client, uri string) (*models.RPKISnapshot, error) {
	return FetchRRDPSnapshot(ctx, c, uri)
}

// rrdpNotificationXML is the wire model for RRDP notification.xml.
type rrdpNotificationXML struct {
	XMLName   xml.Name     `xml:"notification"`
	Version   string       `xml:"version,attr"`
	SessionID string       `xml:"session_id,attr"`
	Serial    string       `xml:"serial,attr"`
	Snapshot  rrdpRefXML   `xml:"snapshot"`
	Deltas    []rrdpRefXML `xml:"delta"`
}

type rrdpRefXML struct {
	Serial string `xml:"serial,attr"`
	URI    string `xml:"uri,attr"`
	Hash   string `xml:"hash,attr"`
}

// parseRRDPNotification decodes an RRDP notification.xml stream.
func parseRRDPNotification(r io.Reader) (*models.RRDPNotification, error) {
	var raw rrdpNotificationXML
	if err := xml.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("RRDP notification XML decode failed: %w", err)
	}
	n := &models.RRDPNotification{
		Version:   raw.Version,
		SessionID: raw.SessionID,
		Snapshot:  models.RRDPRef{URI: raw.Snapshot.URI, Hash: raw.Snapshot.Hash},
	}
	if s, err := strconv.ParseInt(raw.Serial, 10, 64); err == nil {
		n.Serial = s
	}
	if s, err := strconv.ParseInt(raw.Snapshot.Serial, 10, 64); err == nil {
		n.Snapshot.Serial = s
	}
	n.Deltas = make([]models.RRDPRef, 0, len(raw.Deltas))
	for _, d := range raw.Deltas {
		ref := models.RRDPRef{URI: d.URI, Hash: d.Hash}
		if s, err := strconv.ParseInt(d.Serial, 10, 64); err == nil {
			ref.Serial = s
		}
		n.Deltas = append(n.Deltas, ref)
	}
	return n, nil
}

// parseRPKISnapshot streams an RRDP snapshot/delta XML and collects the rsync
// URIs of every <publish> and <withdraw> element, discarding the (large)
// base64 CMS bodies to keep memory bounded. Element local names are matched
// regardless of XML namespace.
func parseRPKISnapshot(r io.Reader) (*models.RPKISnapshot, error) {
	dec := xml.NewDecoder(r)
	snap := &models.RPKISnapshot{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("RPKI snapshot XML stream failed: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			local := localName(t.Name)
			switch local {
			case "snapshot":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "version":
						snap.Version = a.Value
					case "session_id":
						snap.SessionID = a.Value
					case "serial":
						if s, err := strconv.ParseInt(a.Value, 10, 64); err == nil {
							snap.Serial = s
						}
					}
				}
			case "publish":
				if uri := attrLocal(t.Attr, "uri"); uri != "" {
					snap.Published = append(snap.Published, uri)
				}
			case "withdraw":
				if uri := attrLocal(t.Attr, "uri"); uri != "" {
					snap.Withdrawn = append(snap.Withdrawn, uri)
				}
			}
		case xml.CharData:
			// Discard base64 bodies of <publish> elements.
			_ = t
		}
	}
	return snap, nil
}

// localName returns the local part of an xml.Name (the part after any
// namespace prefix). For names without a namespace it returns Space unchanged.
func localName(name xml.Name) string {
	if name.Local != "" {
		return name.Local
	}
	return name.Space
}

// attrLocal returns the value of the first attribute whose local name matches
// key, regardless of namespace.
func attrLocal(attrs []xml.Attr, key string) string {
	for _, a := range attrs {
		if a.Name.Local == key {
			return a.Value
		}
	}
	return ""
}
