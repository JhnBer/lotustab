package connection

import "time"

// Profile is a saved connection (no secrets).
type Profile struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	DriverID  string            `json:"driverId"`
	Params    map[string]string `json:"params"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// Clone returns a shallow copy with a copied params map.
func (p Profile) Clone() Profile {
	out := p
	if p.Params != nil {
		out.Params = make(map[string]string, len(p.Params))
		for k, v := range p.Params {
			out.Params[k] = v
		}
	} else {
		out.Params = map[string]string{}
	}
	return out
}
