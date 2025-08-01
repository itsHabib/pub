package couchbase

// CasManager combines CAS getting and setting capabilities.
type CasManager interface {
	CasGetter
	CasSetter
}

// CasSetter defines the interface for setting CAS values.
// CAS (Compare-And-Swap) values are used for optimistic concurrency control.
type CasSetter interface {
	Set(c uint64)
}

// CasGetter defines the interface for retrieving CAS values.
type CasGetter interface {
	Get() int
}

// Cas provides a simple implementation of CAS value management.
// It can be embedded in structs that need CAS functionality.
type Cas struct {
	c uint64
}

// GetCas returns the current CAS value.
func (c *Cas) GetCas() uint64 {
	return c.c
}

// SetCas updates the CAS value.
func (c *Cas) SetCas(cas uint64) {
	c.c = cas
}
