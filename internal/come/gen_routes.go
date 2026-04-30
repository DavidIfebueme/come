package come

import (
	"fmt"
	"strings"
)

func GenRoutes(proj *Project, feat Feature) string {
	var sb strings.Builder
	sb.WriteString("package " + feat.Name + "\n\nimport(\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString(fmt.Sprintf("func RegisterRoutes(mux *http.ServeMux,h *Handler){\n"))
	for _, r := range feat.Routes {
		pattern := r.Method + " " + goPathPattern(r.Path)
		pascalHandler := toPascalCase(r.Handler)
		sb.WriteString(fmt.Sprintf("\tmux.HandleFunc(%q,h.%s)\n", pattern, pascalHandler))
	}
	sb.WriteString("}\n")

	return sb.String()
}
