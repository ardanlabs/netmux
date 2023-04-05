// Package hosts manages the set of hosts that are provided.
package hosts

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Entry represents a single host entry.
type Entry struct {
	Addr    string
	Names   []string
	Comment string
}

// NewEntry constructs an entry from the specified string. The entry needs
// to be in the format of '10.0.0.1 a.host b.host #hosts1'.
func NewEntry(entry string) Entry {
	parts := strings.Fields(entry)
	if len(parts) < 1 {
		return Entry{}
	}

	e := Entry{
		Addr: parts[0],
	}

	for i := 1; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "#") {
			e.Comment = parts[i][1:]
			break
		}

		e.Names = append(e.Names, parts[i])
	}

	return e
}

// Equals tests that the host is identical to the specified host.
func (e Entry) Equals(ent Entry) bool {
	if e.Addr != ent.Addr {
		return false
	}

	if len(e.Names) != len(ent.Names) {
		return false
	}

	for i := range e.Names {
		if e.Names[i] != ent.Names[i] {
			return false
		}
	}

	return e.Comment == ent.Comment
}

// String implements the Stringer interface.
func (e Entry) String() string {
	return fmt.Sprintf("%s %s %s", e.Addr, strings.Join(e.Names, " "), e.Comment)
}

// =============================================================================

// Hosts represents a collection of host entries.
type Hosts struct {
	fName   string
	entries []Entry
	mu      sync.Mutex
}

// New returns an empty hosts value.
func New() *Hosts {
	return &Hosts{}
}

// Load reads a file of host entries and returns the values.
func Load(fName string) (*Hosts, error) {
	bs, err := os.ReadFile(fName)
	if err != nil {
		return &Hosts{}, fmt.Errorf("os.ReadFile: %w", err)
	}

	h := Hosts{
		fName: fName,
	}

	h.addEntries(bs)

	return &h, nil
}

// Add takes the specified values and adds a new entry to the list.
func (h *Hosts) Add(addr string, names []string, comment string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	e := Entry{
		Addr:    addr,
		Names:   names,
		Comment: comment,
	}

	h.entries = append(h.entries, e)

	if err := h.save(); err != nil {
		return fmt.Errorf("h.save: %w", err)
	}

	return nil
}

// Remove will remove an entry based on the comment.
func (h *Hosts) Remove(comment string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var es []Entry
	for _, e := range h.entries {
		if !strings.Contains(e.Comment, comment) {
			es = append(es, e)
		}
	}

	h.entries = es

	if err := h.save(); err != nil {
		return fmt.Errorf("h.save: %w", err)
	}

	return nil
}

// Equals tests that the hosts is identical to the specified hosts.
func (h *Hosts) Equals(hst *Hosts) bool {
	if len(h.entries) != len(hst.entries) {
		return false
	}

	for i := range h.entries {
		if !h.entries[i].Equals(h.entries[i]) {
			return false
		}
	}

	return true
}

// =============================================================================

func (h *Hosts) addEntries(entries []byte) {
	scn := bufio.NewScanner(bytes.NewReader(entries))

	for scn.Scan() {
		if len(scn.Text()) < 1 {
			continue
		}

		h.entries = append(h.entries, NewEntry(scn.Text()))
	}
}

func (h *Hosts) save() error {
	if err := os.WriteFile(h.fName, h.bytes(), 0600); err != nil {
		return fmt.Errorf("os.WriteFile: %w", err)
	}

	return nil
}

func (h *Hosts) bytes() []byte {
	var buf bytes.Buffer

	for _, e := range h.entries {
		buf.WriteString(e.String())
		buf.WriteString("\n")
	}

	return buf.Bytes()
}
