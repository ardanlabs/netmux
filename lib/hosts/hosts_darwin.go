package hosts

const Fname = "/etc/hosts"

var def *Hosts

func Default() *Hosts {
	if def == nil {
		def = NewFile(Fname)
		err := def.Load()
		if err != nil {
			panic(err)
		}
	}
	return def
}
