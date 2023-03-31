package types

import (
	"strconv"
	"strings"
)

type IdParseable string

func (i IdParseable) Parse() []int {
	parts := strings.Split(string(i), ".")
	ret := make([]int, len(parts))
	for i := 0; i < len(parts); i++ {
		tmpi, err := strconv.Atoi(parts[i])
		if err != nil {
			ret[i] = -1
		}
		ret[i] = tmpi
	}
	return ret
}
func (i IdParseable) Ctx() string {
	parts := strings.Split(string(i), ".")
	return parts[0]
}
func (i IdParseable) Svc() string {
	parts := strings.Split(string(i), ".")
	return parts[1]
}

func (i IdParseable) CtxId() int {
	return i.Parse()[0]
}
func (i IdParseable) SvcId() int {
	return i.Parse()[1]
}
