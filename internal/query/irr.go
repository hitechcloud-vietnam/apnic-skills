package query

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// IRRObjectTypes lists every APNIC IRR (RPSL) database object type published as
// a gzipped dump under https://ftp.apnic.net/apnic/whois/apnic.db.<type>.gz.
// Pass any of these to FetchIRRDatabase.
var IRRObjectTypes = []string{
	"as-block",
	"as-set",
	"aut-num",
	"domain",
	"filter-set",
	"inet6num",
	"inetnum",
	"inet-rtr",
	"irt",
	"key-cert",
	"limerick",
	"mntner",
	"organisation",
	"peering-set",
	"role",
	"route",
	"route6",
	"route-set",
	"rtr-set",
}

// isIRRObjectType reports whether t is a known APNIC IRR object type.
func isIRRObjectType(t string) bool {
	for _, v := range IRRObjectTypes {
		if v == t {
			return true
		}
	}
	return false
}

// FetchIRRDatabase fetches and parses an APNIC IRR database dump for the given
// object type (one of IRRObjectTypes). The dump is gzip-compressed and is
// transparently decompressed by FetchText. objType must be a known type,
// otherwise ErrInvalidArgument is returned.
func FetchIRRDatabase(ctx context.Context, c *transport.Client, objType string) (*models.IRRDatabase, error) {
	if !isIRRObjectType(objType) {
		return nil, fmt.Errorf("%w: %q", transport.ErrInvalidIRRType, objType)
	}
	url := transport.BuildIRRDBURL(c.FTPBaseURL(), objType)
	body, err := c.FetchTextStr(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseIRRDatabase(objType, body)
}

// FetchIRRCurrentSerial fetches the APNIC.CURRENTSERIAL value, which is the
// current serial number of the APNIC IRR database. It is returned as an integer.
func FetchIRRCurrentSerial(ctx context.Context, c *transport.Client) (int64, error) {
	url := transport.BuildIRRCurrentSerialURL(c.FTPBaseURL())
	body, err := c.FetchText(ctx, url)
	if err != nil {
		return 0, err
	}
	serial, err := strconv.ParseInt(strings.TrimSpace(body), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("CURRENTSERIAL parse failed: %w", err)
	}
	return serial, nil
}

// parseIRRDatabase parses an RPSL database dump. Objects are separated by blank
// lines. Within an object, the first line's attribute name is the object type
// and its value is the primary key. Comment lines (starting with '#') are
// skipped. Continuation lines (leading '+' or whitespace) are folded into the
// preceding attribute's value; a leading '+' suppresses the usual extra space.
//
// This parser is independent of ParseWhoisResponse: IRR dumps use the same RPSL
// syntax but are bulk multi-object files, whereas whois responses are typically
// single objects with a different surrounding format.
func parseIRRDatabase(objType, data string) (*models.IRRDatabase, error) {
	db := &models.IRRDatabase{Type: objType, Objects: make([]models.IRRObject, 0, 1024)}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // RPSL lines can be long

	var (
		cur     *models.IRRObject
		lastKey string
	)

	flush := func() {
		if cur != nil {
			db.Objects = append(db.Objects, *cur)
			cur = nil
			lastKey = ""
		}
	}

	for scanner.Scan() {
		raw := scanner.Text()

		// Blank line separates objects.
		if strings.TrimSpace(raw) == "" {
			flush()
			continue
		}

		// Comment lines (outside object continuation context).
		if strings.HasPrefix(strings.TrimLeft(raw, " \t"), "#") {
			continue
		}

		// Continuation line: starts with whitespace or '+', belongs to lastKey.
		trimmed := strings.TrimLeft(raw, " \t")
		if raw != "" && (raw[0] == ' ' || raw[0] == '\t') {
			if cur == nil || lastKey == "" {
				// Continuation with no active attribute; ignore.
				continue
			}
			val := trimmed
			if strings.HasPrefix(val, "+") {
				val = val[1:]
			} else {
				val = " " + val
			}
			cur.Attributes[lastKey] = append(cur.Attributes[lastKey], val)
			continue
		}

		// Attribute line: "key: value".
		colon := strings.Index(raw, ":")
		if colon < 0 {
			// Not an attribute; skip defensively.
			continue
		}
		key := strings.ToLower(strings.TrimSpace(raw[:colon]))
		val := strings.TrimSpace(raw[colon+1:])

		if cur == nil {
			// Start a new object.
			cur = &models.IRRObject{
				Type:       key,
				PrimaryKey: val,
				Attributes: map[string][]string{},
			}
		}
		cur.Attributes[key] = append(cur.Attributes[key], val)
		lastKey = key
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("IRR database scan failed: %w", err)
	}
	return db, nil
}
