package couchbase

type CasManager interface {
	CasGetter
	CasSetter
}

type CasSetter interface {
	Set(c uint64)
}

type CasGetter interface {
	Get() int
}

type Cas struct {
	c uint64
}

func (c *Cas) GetCas() uint64 {
	return c.c
}

func (c *Cas) SetCas(cas uint64) {
	c.c = cas
}
