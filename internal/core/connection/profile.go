package connection

import "time"

const DefaultColor = "blue"

// Palette is the allowed connection color keys (Tailwind hue names).
var Palette = []string{
	"red", "orange", "amber", "green", "teal", "sky", "blue", "indigo", "violet", "pink", "slate",
}

// Profile is a saved connection (no secrets).
type Profile struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	DriverID  string            `json:"driverId"`
	Params    map[string]string `json:"params"`
	Color     string            `json:"color"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// ValidColor reports whether color is a known palette key.
func ValidColor(color string) bool {
	for _, item := range Palette {
		if item == color {
			return true
		}
	}
	return false
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
