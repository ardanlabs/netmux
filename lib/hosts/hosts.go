package hosts

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"sync"
)

type Entry struct {
	Addr    string
	Names   []string
	Comment string
}

func (e *Entry) Equals(f Entry) bool {
	if e.Addr != f.Addr {
		return false
	}
	if len(e.Names) != len(f.Names) {
		return false
	}
	for i := range e.Names {
		if e.Names[i] != f.Names[i] {
			return false
		}
	}
	return e.Comment == f.Comment
}
func (e *Entry) String() string {
	var hosts = strings.Join(e.Names, " ")
	return fmt.Sprintf("%s %s %s", e.Addr, hosts, e.Comment)
}
func (e *Entry) Load(s string) {
	parts := strings.Fields(s)
	if len(parts) < 1 {
		return
	}
	e.Addr = parts[0]
	var i = 1
	for ; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "#") {
			var comment = strings.Join(parts[i:], " ")
			e.Comment = comment
			return
		}
		e.Names = append(e.Names, parts[i])
	}
}
func (e *Entry) CommentMatches(s string) bool {
	return strings.Contains(e.Comment, s)
}

type Hosts struct {
	fname   string
	entries []Entry
	mx      sync.Mutex
}

func (m *Hosts) LoadBytes(bs []byte) {
	fileScanner := bufio.NewScanner(bytes.NewReader(bs))
	for fileScanner.Scan() {
		if len(fileScanner.Text()) < 1 {
			continue
		}
		var ae Entry
		ae.Load(fileScanner.Text())
		m.entries = append(m.entries, ae)
	}
}
func (m *Hosts) Bytes() []byte {
	var buf = &bytes.Buffer{}
	for _, e := range m.entries {
		l := e.String()
		buf.WriteString(l)
		buf.WriteString("\n")
	}
	return buf.Bytes()
}
func (m *Hosts) RemoveByComment(c string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	var ne []Entry
	for _, v := range m.entries {
		if !v.CommentMatches(c) {
			ne = append(ne, v)
		} else {
			logrus.Debugf("Removing hosts entry: %s", v.String())
		}
	}
	m.entries = ne
	return m.unSyncSave()

}
func (m *Hosts) Equals(n *Hosts) bool {
	if len(m.entries) != len(n.entries) {
		return false
	}
	for i := range m.entries {
		if !m.entries[i].Equals(n.entries[i]) {
			return false
		}
	}
	return true
}
func (m *Hosts) Load() error {
	m.mx.Lock()
	defer m.mx.Unlock()
	logrus.Debugf("Loading hosts from %s", m.fname)
	bs, err := os.ReadFile(m.fname)
	if err != nil {
		return err
	}
	m.LoadBytes(bs)
	return nil

}

func (m *Hosts) unSyncSave() error {
	//logrus.Debugf("Saving hosts to %s", m.fname)
	err := os.WriteFile(m.fname, m.Bytes(), 0600)
	if err != nil {
		return err
	}
	return nil
}
func (m *Hosts) Add(adr string, names []string, comment string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	e := Entry{
		Addr:    adr,
		Names:   names,
		Comment: comment,
	}
	logrus.Debugf("Adding hosts entry: %s", e.String())
	m.entries = append(m.entries, e)
	return m.unSyncSave()
}

func New() *Hosts {
	return new(Hosts)
}

func NewFile(f string) *Hosts {
	var ret = new(Hosts)
	ret.fname = f
	return ret
}
