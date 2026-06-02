package dns

import (
	"fmt"

	"codeberg.org/miekg/dns"
)

// RecordType identifies a DNS resource record type (A, AAAA, CNAME, ...). It is
// a thin wrapper over the underlying dns library's numeric type codes so that
// callers of this package never need to import that library directly.
type RecordType uint16

// The record types this package understands. Their numeric values match the
// IANA-assigned DNS type codes.
const (
	TypeA     = RecordType(dns.TypeA)
	TypeAAAA  = RecordType(dns.TypeAAAA)
	TypeCNAME = RecordType(dns.TypeCNAME)
	TypeMX    = RecordType(dns.TypeMX)
	TypeNS    = RecordType(dns.TypeNS)
	TypeTXT   = RecordType(dns.TypeTXT)
	TypeSOA   = RecordType(dns.TypeSOA)
	TypePTR   = RecordType(dns.TypePTR)
	TypeSRV   = RecordType(dns.TypeSRV)
)

// String returns the canonical mnemonic for the record type (e.g. "A",
// "CNAME"). Unknown types render as the empty string.
func (rt RecordType) String() string {
	return dns.TypeToString[uint16(rt)]
}

// Record is a single resolved DNS record. Value holds the type-specific data
// rendered as a string (an IP for A/AAAA, the target name for CNAME, and so
// on), keeping the type usable without depending on the wire-format library.
type Record struct {
	Type  RecordType
	Name  string
	Value string
	TTL   uint32
}

// String renders the record in a human-readable "name type: value (TTL: n)"
// form for logging.
func (r Record) String() string {
	return fmt.Sprintf("%s %s: %s (TTL: %d)", r.Name, r.Type.String(), r.Value, r.TTL)
}
