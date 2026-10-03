package query

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
	"github.com/hitechcloud-vietnam/apnic-skills/internal/transport"
)

// QueryWhois performs a raw Whois query against the APNIC Whois server.
// Returns the raw Whois response text.
func QueryWhois(ctx context.Context, c *transport.Client, query string) (string, error) {
	return queryWhois(ctx, c, query)
}

// QueryWhoisIP performs a Whois query for an IP address.
// This is a convenience method that queries and parses the result.
func QueryWhoisIP(ctx context.Context, c *transport.Client, ip string) (*models.WhoisInfo, error) {
	raw, err := queryWhois(ctx, c, ip)
	if err != nil {
		return nil, err
	}
	info := ParseWhoisResponse(raw)
	return &info, nil
}

// QueryWhoisASN performs a Whois query for an Autonomous System Number.
// asn should be a plain number (e.g. 13335), not "AS13335".
func QueryWhoisASN(ctx context.Context, c *transport.Client, asn int64) (*models.WhoisInfo, error) {
	query := fmt.Sprintf("AS%d", asn)
	raw, err := queryWhois(ctx, c, query)
	if err != nil {
		return nil, err
	}
	info := ParseWhoisResponse(raw)
	return &info, nil
}

// QueryWhoisWithFlags performs a Whois query with additional flags.
// Common flags: "B" (brief), "r" (no recursion), "l" (one level less specific).
func QueryWhoisWithFlags(ctx context.Context, c *transport.Client, query string, flags string) (string, error) {
	if flags != "" {
		query = flags + " " + query
	}
	return queryWhois(ctx, c, query)
}

// queryWhois performs the actual TCP Whois query.
func queryWhois(ctx context.Context, c *transport.Client, query string) (string, error) {
	// Apply the same anti-scraping pacing to whois as to HTTP so high-frequency
	// whois queries don't trip rate limits. Whois has no browser headers (it is
	// a plain TCP protocol), but jitter + rate limiting still apply.
	if err := c.WaitRateLimit(ctx); err != nil {
		return "", err
	}
	c.Jitter(ctx)

	var conn net.Conn
	var err error

	if d := c.DialWhois(); d != nil {
		conn, err = d(ctx, "tcp", c.WhoisServer())
	} else {
		dialer := net.Dialer{Timeout: c.WhoisTimeout()}
		conn, err = dialer.DialContext(ctx, "tcp", c.WhoisServer())
	}
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	query = strings.TrimSpace(query) + "\r\n"
	if _, err := conn.Write([]byte(query)); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, conn); err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	return buf.String(), nil
}

// ParseWhoisResponse parses a raw Whois response into a structured WhoisInfo.
//
// A real APNIC whois response for an IP is a concatenation of several RPSL
// objects separated by blank lines: the primary inetnum/inet6num object, plus
// secondary irt/organisation/role/route objects. This returns the first primary
// object with CIDR/OriginASN supplemented from any route object and OrgName from
// any organisation object, delegating the multi-object walk to
// ParseWhoisResponseList and taking element [0]. For queries that return
// multiple primary objects (e.g. "-L" all-less-specific), use
// ParseWhoisResponseList to get the full list. Returns a zero-value WhoisInfo
// when no primary object is found.
func ParseWhoisResponse(response string) models.WhoisInfo {
	list := ParseWhoisResponseList(response)
	if len(list) == 0 {
		return models.WhoisInfo{CIDR: []string{}}
	}
	return list[0]
}

// ParseWhoisResponseList parses a raw Whois response that may contain multiple
// primary objects (e.g. the response to a "-L" all-less-specific or "-M"
// all-more-specific query, which returns several inetnum/route objects). Each
// primary object block produces one WhoisInfo, in document order. Secondary
// objects (irt/organisation/role) are folded into the nearest preceding primary
// object's CIDR/OriginASN/OrgName supplements, matching ParseWhoisResponse's
// single-object semantics. Returns an empty slice (not nil) when no primary
// object is found, so callers always get a valid range.
func ParseWhoisResponseList(response string) []models.WhoisInfo {
	result := []models.WhoisInfo{}
	blocks := splitWhoisBlocks(response)

	// Track the index of the current primary object so secondary-object
	// supplements (route CIDR, org-name) attach to it rather than spawning a
	// new entry.
	currentIdx := -1
	for _, block := range blocks {
		kv := parseWhoisBlock(block)
		if len(kv) == 0 {
			continue
		}

		isPrimary := false
		// inetnum/inet6num/aut-num always start a new primary entry. route and
		// as-block start a new entry only when no primary is in progress;
		// otherwise they fold into the preceding primary as a supplement
		// (matching ParseWhoisResponse's single-object semantics, where a
		// trailing route block feeds the inetnum's CIDR/origin rather than
		// spawning a second entry).
		for _, key := range []string{"inetnum", "inet6num", "aut-num"} {
			if _, ok := kv[key]; ok {
				isPrimary = true
				break
			}
		}
		if !isPrimary {
			for _, key := range []string{"route", "as-block"} {
				if _, ok := kv[key]; ok && currentIdx < 0 {
					isPrimary = true
					break
				}
			}
		}

		if isPrimary {
			info := models.WhoisInfo{CIDR: []string{}}
			if v, ok := kv["inetnum"]; ok {
				info.Network = v
			} else if v, ok := kv["inet6num"]; ok {
				info.Network = v
			} else if v, ok := kv["aut-num"]; ok {
				info.Network = v
			} else if v, ok := kv["route"]; ok {
				info.Network = v
			} else if v, ok := kv["as-block"]; ok {
				info.Network = v
			}
			if v, ok := kv["netname"]; ok {
				info.NetName = v
			}
			if v, ok := kv["country"]; ok {
				info.Country = v
			}
			if v, ok := kv["status"]; ok {
				info.Status = v
			}
			if v, ok := kv["descr"]; ok && info.OrgName == "" {
				info.OrgName = v
			}
			if v, ok := kv["abuse-c"]; ok {
				info.AbuseContact = v
			}
			if v, ok := kv["abuse-mailbox"]; ok && info.AbuseMailbox == "" {
				info.AbuseMailbox = v
			}
			if v, ok := kv["parent"]; ok {
				info.Parent = v
			}
			if v, ok := kv["created"]; ok {
				if t, err := parseWhoisDate(v); err == nil {
					info.Created = t
				}
			}
			if v, ok := kv["last-modified"]; ok {
				if t, err := parseWhoisDate(v); err == nil {
					info.LastUpdated = t
				}
			}
			// A route-as-primary block also feeds its own CIDR/origin.
			if v, ok := kv["route"]; ok {
				info.CIDR = appendCIDR(info.CIDR, v)
			}
			if v, ok := kv["origin"]; ok && info.OriginASN == "" {
				info.OriginASN = v
			}
			result = append(result, info)
			currentIdx = len(result) - 1
			continue
		}

		// Secondary object: supplement the current primary object's CIDR /
		// OriginASN / OrgName if one is in progress.
		if currentIdx >= 0 {
			cur := &result[currentIdx]
			if v, ok := kv["route"]; ok {
				cur.CIDR = appendCIDR(cur.CIDR, v)
			}
			if v, ok := kv["origin"]; ok && cur.OriginASN == "" {
				cur.OriginASN = v
			}
			if v, ok := kv["org-name"]; ok && cur.OrgName == "" {
				cur.OrgName = v
			}
			if v, ok := kv["organisation"]; ok && cur.OrgName == "" {
				cur.OrgName = v
			}
			if v, ok := kv["abuse-mailbox"]; ok && cur.AbuseMailbox == "" {
				cur.AbuseMailbox = v
			}
		}
	}

	return result
}

// splitWhoisBlocks splits a raw whois response into RPSL object blocks on blank
// lines, stripping comment lines (% or #) within each block.
func splitWhoisBlocks(response string) []string {
	var blocks []string
	var current []string
	for _, raw := range strings.Split(response, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
			}
			continue
		}
		if strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	return blocks
}

// parseWhoisBlock parses a single RPSL object block into a key→value map. Only
// the first value of a repeated key is kept (e.g. the first descr line), since
// the structured model holds single strings. Multi-valued keys like route are
// handled by the caller scanning for that key across blocks.
func parseWhoisBlock(block string) map[string]string {
	kv := make(map[string]string)
	for _, line := range strings.Split(block, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, exists := kv[key]; !exists {
			kv[key] = value
		}
	}
	return kv
}

// appendCIDR appends a CIDR string to the list if not already present (a single
// inetnum may map to multiple route objects; dedupe keeps the list clean).
func appendCIDR(list []string, cidr string) []string {
	for _, c := range list {
		if c == cidr {
			return list
		}
	}
	return append(list, cidr)
}

// parseWhoisDate attempts to parse a date string from Whois responses.
func parseWhoisDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"20060102",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}
