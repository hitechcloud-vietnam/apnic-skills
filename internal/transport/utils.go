package transport

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/models"
)

// ParseIPv4Count parses an IPv4 host count value.
func ParseIPv4Count(s string) (int64, error) {
	count, err := strconv.ParseInt(s, 10, 64)
	if err != nil || count <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidIP, s)
	}
	return count, nil
}

// ParseIPv6Prefix parses an IPv6 prefix length value.
func ParseIPv6Prefix(s string) (int64, error) {
	prefix, err := strconv.ParseInt(s, 10, 64)
	if err != nil || prefix < 0 || prefix > 128 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidIP, s)
	}
	return prefix, nil
}

// ParseASNValue parses an ASN value from delegated stats.
// In delegated stats files, ASN values are plain integers (not "AS" prefixed).
func ParseASNValue(s string) (int64, error) {
	asn, err := strconv.ParseInt(s, 10, 64)
	if err != nil || asn < 0 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidASN, s)
	}
	return asn, nil
}

// ParseASNCount parses an ASN count value from delegated stats.
func ParseASNCount(s string) (int64, error) {
	count, err := strconv.ParseInt(s, 10, 64)
	if err != nil || count <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidASN, s)
	}
	return count, nil
}

// ParseStatsHeader parses the version/header line of a stats file.
// Format: version|registry|serial|records|startdate|enddate|UTCoffset
func ParseStatsHeader(line string) (*models.StatsFileHeader, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 7 {
		return nil, fmt.Errorf("%w: invalid header line", ErrInvalidStatsType)
	}

	serial, _ := strconv.ParseInt(parts[2], 10, 64)
	records, _ := strconv.ParseInt(parts[3], 10, 64)
	utcOffset, _ := strconv.Atoi(parts[6])

	var startDate, endDate time.Time
	if d, err := time.Parse("20060102", parts[4]); err == nil {
		startDate = d
	}
	if d, err := time.Parse("20060102", parts[5]); err == nil {
		endDate = d
	}

	return &models.StatsFileHeader{
		Version:   parts[0],
		Registry:  parts[1],
		Serial:    serial,
		Records:   records,
		StartDate: startDate,
		EndDate:   endDate,
		UTCOffset: utcOffset,
	}, nil
}

// ParseSummaryLine parses a summary line from a stats file.
// Format: registry|*|type|*|count|summary
func ParseSummaryLine(line string) (*models.StatsSummary, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 6 || parts[5] != "summary" {
		return nil, fmt.Errorf("%w: invalid summary line", ErrInvalidStatsType)
	}

	count, _ := strconv.ParseInt(parts[4], 10, 64)

	return &models.StatsSummary{
		Registry: parts[0],
		Type:     parts[2],
		Count:    count,
	}, nil
}

// IsSummaryLine checks if a line is a summary line.
func IsSummaryLine(line string) bool {
	parts := strings.Split(line, "|")
	return len(parts) >= 6 && parts[5] == "summary"
}

// IsHeaderLine checks if a line is a header (version) line.
func IsHeaderLine(line string) bool {
	parts := strings.Split(line, "|")
	if len(parts) < 7 {
		return false
	}
	// Version lines start with a version number like "2" or "2.3"
	v := parts[0]
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return true
	}
	return false
}

// statsFileName returns the base file name for a stats data file of the given
// type and date, using APNIC's naming convention:
//   - latest:  {type}-apnic-latest   (e.g. delegated-apnic-latest)
//   - dated:   delegated-apnic-{date}, delegated-apnic-extended-{date},
//     delegated-apnic-ipv6-assigned-{date}, assigned-apnic-{date},
//     legacy-apnic-{date}
func statsFileName(dataType, date string) string {
	if date == "" {
		if dataType == "delegated" || strings.HasPrefix(dataType, "delegated-") {
			variant := strings.TrimPrefix(dataType, "delegated-")
			if variant == "delegated" {
				return "delegated-apnic-latest"
			}
			return "delegated-apnic-" + variant + "-latest"
		}
		return dataType + "-apnic-latest"
	}
	if dataType == "delegated" || strings.HasPrefix(dataType, "delegated-") {
		variant := strings.TrimPrefix(dataType, "delegated-")
		if variant == "delegated" {
			return "delegated-apnic-" + date
		}
		return "delegated-apnic-" + variant + "-" + date
	}
	return dataType + "-apnic-" + date
}

// BuildStatsURL constructs the URL for a stats data file.
// For the latest file (date == "") the file lives at the stats root.
// For a specific date (YYYYMMDD) the file lives under a {year}/ subdirectory and
// is gzip-compressed with a ".gz" suffix.
// dataType: "delegated", "delegated-extended", "delegated-ipv6-assigned",
// "assigned", "legacy".
func BuildStatsURL(baseURL, dataType, date string) string {
	name := statsFileName(dataType, date)
	if date == "" {
		return baseURL + name
	}
	return fmt.Sprintf("%s%s/%s.gz", baseURL, date[:4], name)
}

// BuildStatsMD5URL constructs the URL for a stats file MD5 checksum.
// Latest checksums are plain text; dated checksums are gzip-compressed (.md5.gz).
func BuildStatsMD5URL(baseURL, dataType, date string) string {
	name := statsFileName(dataType, date)
	if date == "" {
		return baseURL + name + ".md5"
	}
	return fmt.Sprintf("%s%s/%s.md5.gz", baseURL, date[:4], name)
}

// BuildStatsASCURL constructs the URL for a stats file ASC (PGP) signature.
// Latest signatures are plain text; dated signatures are gzip-compressed (.asc.gz).
func BuildStatsASCURL(baseURL, dataType, date string) string {
	name := statsFileName(dataType, date)
	if date == "" {
		return baseURL + name + ".asc"
	}
	return fmt.Sprintf("%s%s/%s.asc.gz", baseURL, date[:4], name)
}

// ParseOpaqueID extracts the opaque-id from a stats record.
func ParseOpaqueID(s string) string {
	return strings.TrimSpace(s)
}

// BuildTransfersAllURL constructs the URL for the cumulative transfers-all log.
// date == "" fetches the latest (transfer-all-apnic-latest); a YYYYMMDD date
// fetches the archived daily snapshot under transfers-all/apnic/{YYYY}/. The
// data lives under the FTP root (not the stats subdirectory), so it uses
// ftpBaseURL.
func BuildTransfersAllURL(ftpBaseURL, date string) string {
	if date == "" {
		return ftpBaseURL + "transfers-all/apnic/transfer-all-apnic-latest"
	}
	// date is YYYYMMDD; the year prefix is the first 4 chars.
	return fmt.Sprintf("%stransfers-all/apnic/%s/transfer-all-apnic-%s", ftpBaseURL, date[:4], date)
}

// BuildTransfersAllSidecarURL constructs the URL for the .md5 or .asc sidecar of
// the cumulative transfers-all log. suffix is ".md5" or ".asc".
func BuildTransfersAllSidecarURL(ftpBaseURL, date, suffix string) string {
	return BuildTransfersAllURL(ftpBaseURL, date) + suffix
}

// BuildTelemetryURL constructs the URL for the whois-rdap-stats telemetry JSON.
// date == "" fetches the latest; a YYYYMMDD date fetches the archived snapshot.
func BuildTelemetryURL(ftpBaseURL, date string) string {
	if date == "" {
		return ftpBaseURL + "apnic/whois-rdap-stats/whois-rdap-stats.json"
	}
	return fmt.Sprintf("%s/apnic/whois-rdap-stats/%s/whois-rdap-stats-%s.json", ftpBaseURL, date[:4], date)
}

// BuildTelemetrySidecarURL constructs the .md5 sidecar URL for the telemetry.
func BuildTelemetrySidecarURL(ftpBaseURL, date string) string {
	return BuildTelemetryURL(ftpBaseURL, date) + ".md5"
}

// BuildIRRDBURL constructs the URL for an APNIC IRR database dump. objType is
// one of the IRRObjectTypes (e.g. "inetnum"). The dumps are gzip-compressed.
func BuildIRRDBURL(ftpBaseURL, objType string) string {
	return fmt.Sprintf("%sapnic/whois/apnic.db.%s.gz", ftpBaseURL, objType)
}

// BuildIRRCurrentSerialURL constructs the URL for the APNIC.CURRENTSERIAL file,
// which holds the current IRR database serial number.
func BuildIRRCurrentSerialURL(ftpBaseURL string) string {
	return ftpBaseURL + "apnic/whois/APNIC.CURRENTSERIAL"
}

// BuildThymeURL constructs the URL for an APNIC thyme BGP analysis file.
// source is one of "current", "au", or "hk"; an empty source defaults to
// "current" for backward compatibility. file is one of "data-summary",
// "data-raw-table", "data-badpfx-nos", "data-pfx-nos", "data-used-autnums",
// "data-spar", or "data-singlepfx".
func BuildThymeURL(thymeBaseURL, source, file string) string {
	if source == "" {
		source = "current"
	}
	return strings.TrimRight(thymeBaseURL, "/") + "/" + source + "/" + file
}

// SourceOrDefault returns source if non-empty, else def. Used by thyme Fetch
// methods to let a per-call source override the client's default thymeSource.
func SourceOrDefault(source, def string) string {
	if source != "" {
		return source
	}
	return def
}

// BuildRRDPNotificationURL constructs the URL for the RRDP notification file.
// The default rrdpBaseURL is https://rrdp.apnic.net.
func BuildRRDPNotificationURL(rrdpBaseURL string) string {
	return strings.TrimRight(rrdpBaseURL, "/") + "/notification.xml"
}

// ParseStatsDate parses a date string in YYYYMMDD format.
func ParseStatsDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "00000000" {
		return time.Time{}, nil
	}
	t, err := time.Parse("20060102", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidDate, s)
	}
	return t, nil
}

// FormatDate formats a time.Time to YYYYMMDD string for use in stats file URLs.
func FormatDate(t time.Time) string {
	return t.Format("20060102")
}

// ParseDate parses a YYYYMMDD date string.
func ParseDate(s string) (time.Time, error) {
	return ParseStatsDate(s)
}
