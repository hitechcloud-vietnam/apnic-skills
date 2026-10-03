package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hitechcloud-vietnam/apnic-skills/internal/testutil"
)

func TestDelegatedEntry(t *testing.T) {
	e := DelegatedEntry{
		Registry: "apnic",
		Country:  "AU",
		Type:     "ipv4",
		Start:    "1.0.0.0",
		Value:    256,
		Date:     time.Date(2011, 8, 11, 0, 0, 0, 0, time.UTC),
		Status:   "assigned",
	}
	if e.Registry != "apnic" {
		t.Errorf("registry = %q, want apnic", e.Registry)
	}
	if e.Value != 256 {
		t.Errorf("value = %d, want 256", e.Value)
	}
}

func TestDelegatedExtendedEntry(t *testing.T) {
	e := DelegatedExtendedEntry{
		Registry: "apnic",
		Country:  "CN",
		Type:     "ipv4",
		Start:    "1.0.1.0",
		Value:    256,
		Date:     time.Date(2011, 4, 14, 0, 0, 0, 0, time.UTC),
		Status:   "allocated",
		OpaqueID: "A92E1062",
	}
	if e.OpaqueID != "A92E1062" {
		t.Errorf("opaqueID = %q, want A92E1062", e.OpaqueID)
	}
}

func TestTransferRecord(t *testing.T) {
	r := TransferRecord{
		Type:      "RESOURCE_TRANSFER",
		SourceRIR: "APNIC",
		IPv4Nets: &TransferNetSet{TransferSet: []NetRange{
			{StartAddress: "1.2.3.0", EndAddress: "1.2.3.255"},
		}},
	}
	if r.Type != "RESOURCE_TRANSFER" {
		t.Errorf("type = %q", r.Type)
	}
	if len(r.IPv4Nets.TransferSet) != 1 {
		t.Errorf("transfer set length = %d", len(r.IPv4Nets.TransferSet))
	}
}

func TestOrganization(t *testing.T) {
	org := Organization{Name: "Test Org", CountryCode: "AU"}
	if org.Name != "Test Org" {
		t.Errorf("name = %q", org.Name)
	}
}

func TestChangeRecord(t *testing.T) {
	r := ChangeRecord{
		Country:   "IN",
		Custodian: "A91ED89F",
		Resources: []string{"160.236.32.0/23"},
		Status:    "allocated",
		Type:      "delegated",
	}
	if len(r.Resources) != 1 {
		t.Errorf("resources length = %d", len(r.Resources))
	}
}

func TestWhoisInfo(t *testing.T) {
	info := WhoisInfo{
		Network: "1.1.1.0 - 1.1.1.255",
		Country: "AU",
	}
	if info.Network != "1.1.1.0 - 1.1.1.255" {
		t.Errorf("network = %q", info.Network)
	}
}

func TestStatsFileHeader(t *testing.T) {
	h := StatsFileHeader{
		Version:  "2",
		Registry: "apnic",
		Serial:   20260627,
		Records:  88485,
	}
	if h.Version != "2" {
		t.Errorf("version = %q", h.Version)
	}
}

func TestRDAPNetworkJSON(t *testing.T) {
	var network RDAPNetwork
	err := json.Unmarshal([]byte(testutil.SampleRDAPNetworkJSON), &network)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if network.Name != "APNIC-LABS" {
		t.Errorf("name = %q", network.Name)
	}
	if network.Country != "AU" {
		t.Errorf("country = %q", network.Country)
	}
	if network.IPVersion != "v4" {
		t.Errorf("ipVersion = %q", network.IPVersion)
	}
	if len(network.CIDR0CIDRs) != 1 {
		t.Errorf("cidr0_cidrs length = %d", len(network.CIDR0CIDRs))
	}
	if network.CIDR0CIDRs[0].V4Prefix != "1.1.1.0" || network.CIDR0CIDRs[0].Length != 24 {
		t.Errorf("cidr0 = %+v", network.CIDR0CIDRs[0])
	}
}

func TestRDAPAutnumJSON(t *testing.T) {
	var autnum RDAPAutnum
	err := json.Unmarshal([]byte(testutil.SampleRDAPAutnumJSON), &autnum)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if autnum.Name != "CLOUDFLARE" {
		t.Errorf("name = %q", autnum.Name)
	}
	if autnum.StartAutnum != 13335 {
		t.Errorf("startAutnum = %d", autnum.StartAutnum)
	}
}

func TestRDAPDomainJSON(t *testing.T) {
	var domain RDAPDomain
	err := json.Unmarshal([]byte(testutil.SampleRDAPDomainJSON), &domain)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if domain.LDHName != "1.0.0.1.in-addr.arpa" {
		t.Errorf("ldhName = %q", domain.LDHName)
	}
	if len(domain.Nameservers) != 1 {
		t.Errorf("nameservers length = %d", len(domain.Nameservers))
	}
}

func TestRDAPEntityJSON(t *testing.T) {
	var entity RDAPEntity
	err := json.Unmarshal([]byte(testutil.SampleRDAPEntityJSON), &entity)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if entity.Handle != "AIC3-AP" {
		t.Errorf("handle = %q", entity.Handle)
	}
	if len(entity.Roles) != 2 {
		t.Errorf("roles length = %d", len(entity.Roles))
	}
	if len(entity.Links) != 1 {
		t.Errorf("links length = %d", len(entity.Links))
	}
}

func TestRDAPErrorJSON(t *testing.T) {
	var rdapErr RDAPError
	err := json.Unmarshal([]byte(testutil.SampleRDAPNotFoundJSON), &rdapErr)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if rdapErr.ErrorCode != 404 {
		t.Errorf("errorCode = %d, want 404", rdapErr.ErrorCode)
	}
	if rdapErr.Title != "Not Found" {
		t.Errorf("title = %q", rdapErr.Title)
	}
}

func TestRDAPSearchResultJSON(t *testing.T) {
	var result RDAPSearchResult
	err := json.Unmarshal([]byte(testutil.SampleRDAPSearchJSON), &result)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if len(result.EntitySearchResults) != 2 {
		t.Errorf("entitySearchResults length = %d, want 2", len(result.EntitySearchResults))
	}
	if result.EntitySearchResults[0].Handle != "AIC3-AP" {
		t.Errorf("first result handle = %q, want AIC3-AP", result.EntitySearchResults[0].Handle)
	}
}

func TestDelegatedResult(t *testing.T) {
	r := DelegatedResult{
		Header:    StatsFileHeader{Version: "2"},
		Summaries: []StatsSummary{{Registry: "apnic", Type: "asn", Count: 100}},
		Entries:   []DelegatedEntry{{Country: "AU"}},
	}
	if r.Header.Version != "2" {
		t.Errorf("version = %q", r.Header.Version)
	}
	if len(r.Summaries) != 1 {
		t.Errorf("summaries length = %d", len(r.Summaries))
	}
}

func TestTransfersResult(t *testing.T) {
	r := TransfersResult{
		Metadata:  TransfersMetadata{Producer: "APNIC"},
		Transfers: []TransferRecord{{Type: "RESOURCE_TRANSFER"}},
	}
	if r.Metadata.Producer != "APNIC" {
		t.Errorf("producer = %q", r.Metadata.Producer)
	}
}

func TestChangesResult(t *testing.T) {
	r := ChangesResult{
		Metadata: ChangesMetadata{Count: 3},
		Changes:  []ChangeRecord{{Country: "IN"}},
	}
	if r.Metadata.Count != 3 {
		t.Errorf("count = %d", r.Metadata.Count)
	}
}

func TestAssignedEntry(t *testing.T) {
	e := AssignedEntry{
		Registry: "apnic",
		Country:  "ae",
		Type:     "ipv4",
		Prefix:   "256",
		Count:    12,
		Status:   "assigned",
	}
	if e.Prefix != "256" {
		t.Errorf("prefix = %q", e.Prefix)
	}
}

func TestLegacyEntry(t *testing.T) {
	e := LegacyEntry{
		Registry: "apnic",
		Country:  "",
		Type:     "ipv4",
		Start:    "128.134.0.0",
		Value:    65536,
		Status:   "allocated",
	}
	if e.Country != "" {
		t.Errorf("country should be empty for legacy, got %q", e.Country)
	}
}

func TestASNRange(t *testing.T) {
	r := ASNRange{StartASN: 64512, EndASN: 64520}
	if r.StartASN != 64512 {
		t.Errorf("startASN = %d", r.StartASN)
	}
}

func TestNetRange(t *testing.T) {
	r := NetRange{StartAddress: "1.2.3.0", EndAddress: "1.2.3.255"}
	if r.StartAddress != "1.2.3.0" {
		t.Errorf("startAddress = %q", r.StartAddress)
	}
}

func TestRDAPResponse(t *testing.T) {
	resp := RDAPResponse{
		Conformance: []string{"rdap_level_0"},
		Port43:      "whois.apnic.net",
	}
	if len(resp.Conformance) != 1 {
		t.Errorf("conformance length = %d", len(resp.Conformance))
	}
}

func TestRDAPNotice(t *testing.T) {
	n := RDAPNotice{
		Title:       "Terms of Use",
		Description: []string{"Test"},
	}
	if n.Title != "Terms of Use" {
		t.Errorf("title = %q", n.Title)
	}
}

func TestRDAPLink(t *testing.T) {
	l := RDAPLink{
		Value: "https://example.com",
		Rel:   "self",
		Href:  "https://example.com",
		Type:  "application/rdap+json",
	}
	if l.Rel != "self" {
		t.Errorf("rel = %q", l.Rel)
	}
}

func TestRDAPEvent(t *testing.T) {
	e := RDAPEvent{
		EventAction: "registration",
		EventDate:   "2011-08-10T23:12:35Z",
	}
	if e.EventAction != "registration" {
		t.Errorf("eventAction = %q", e.EventAction)
	}
}

func TestRDAPRemark(t *testing.T) {
	r := RDAPRemark{
		Title:       "Note",
		Description: []string{"Test remark"},
	}
	if r.Title != "Note" {
		t.Errorf("title = %q", r.Title)
	}
}

func TestCIDR0(t *testing.T) {
	c := CIDR0{V4Prefix: "1.1.1.0", Length: 24}
	if c.V4Prefix != "1.1.1.0" {
		t.Errorf("v4prefix = %q", c.V4Prefix)
	}
	if c.Length != 24 {
		t.Errorf("length = %d", c.Length)
	}
}

func TestRDAPNameserver(t *testing.T) {
	ns := RDAPNameserver{
		LDHName: "ns1.example.com",
		IPs:     []string{"1.2.3.4"},
	}
	if ns.LDHName != "ns1.example.com" {
		t.Errorf("ldhName = %q", ns.LDHName)
	}
}

func TestTransfersMetadata(t *testing.T) {
	m := TransfersMetadata{
		Producer:     "APNIC",
		StatsVersion: "4.0",
	}
	if m.Producer != "APNIC" {
		t.Errorf("producer = %q", m.Producer)
	}
}

func TestChangesMetadata(t *testing.T) {
	m := ChangesMetadata{
		Count:      3,
		StatsBegin: "start",
		StatsEnd:   "end",
		Version:    "0.1",
	}
	if m.Count != 3 {
		t.Errorf("count = %d", m.Count)
	}
}

func TestExtendedResult(t *testing.T) {
	r := ExtendedResult{
		Entries: []DelegatedExtendedEntry{{OpaqueID: "A1"}},
	}
	if len(r.Entries) != 1 {
		t.Errorf("entries length = %d", len(r.Entries))
	}
}

func TestAssignedResult(t *testing.T) {
	r := AssignedResult{
		Entries: []AssignedEntry{{Prefix: "256"}},
	}
	if len(r.Entries) != 1 {
		t.Errorf("entries length = %d", len(r.Entries))
	}
}

func TestLegacyResult(t *testing.T) {
	r := LegacyResult{
		Entries: []LegacyEntry{{Start: "128.134.0.0"}},
	}
	if len(r.Entries) != 1 {
		t.Errorf("entries length = %d", len(r.Entries))
	}
}

func TestStatsSummary(t *testing.T) {
	s := StatsSummary{Registry: "apnic", Type: "asn", Count: 100}
	if s.Type != "asn" {
		t.Errorf("type = %q", s.Type)
	}
}

func TestTransferASNSet(t *testing.T) {
	s := TransferASNSet{TransferSet: []ASNRange{{StartASN: 1, EndASN: 10}}}
	if len(s.TransferSet) != 1 {
		t.Errorf("transfer set length = %d", len(s.TransferSet))
	}
}

func TestTransferNetSet(t *testing.T) {
	s := TransferNetSet{TransferSet: []NetRange{{StartAddress: "1.0.0.0"}}}
	if len(s.TransferSet) != 1 {
		t.Errorf("transfer set length = %d", len(s.TransferSet))
	}
}
