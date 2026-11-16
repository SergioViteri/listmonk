package core

import (
	"strings"

	"github.com/lib/pq"
)

// zacaExcludeTags normaliza la lista de etiquetas a excluir de un listado o de
// un borrado por consulta de campañas: sin vacías ni repetidas, y nunca nil
// (el SQL compara con CARDINALITY, que no admite NULL).
func zacaExcludeTags(tags []string) pq.StringArray {
	out := pq.StringArray{}
	seen := map[string]bool{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
