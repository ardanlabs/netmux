package hosts

import (
	"log"
	"testing"
)

func TestBasic(t *testing.T) {
	str := `
127.0.0.1 localhost mbp #localhost
10.0.0.1 a.host b.host #hosts1
10.0.0.2 a.host b.host #hosts2
`
	var a = New()
	var b = New()
	a.LoadBytes([]byte(str))
	log.Printf("%#v", a.entries)
	log.Print(string(a.Bytes()))
	b.LoadBytes(a.Bytes())
	if !a.Equals(b) {
		t.Fatal("Should be equal")
	}
	b.RemoveByComment("hosts2")
	log.Print(string(b.Bytes()))
}
