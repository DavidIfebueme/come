package come

import (
	"fmt"
	"strings"
)

func GenTypes(proj *Project, feat Feature) string {
	var sb strings.Builder
	sb.WriteString("package " + feat.Name + "\n")

	imports := map[string]bool{}
	for _, m := range feat.Manifests {
		if needsTimeImport(&m) {
			imports["time"] = true
		}
		if needsJSONImport(&m) {
			imports["encoding/json"] = true
		}
	}

	if len(imports) > 0 {
		sb.WriteString("\nimport(\n")
		for imp := range imports {
			sb.WriteString(fmt.Sprintf("\t%q\n", imp))
		}
		sb.WriteString(")\n")
	}

	for _, m := range feat.Manifests {
		sb.WriteString(genModelStruct(&m))
		sb.WriteString("\n")
	}

	for _, e := range feat.Enums {
		sb.WriteString(genEnumConsts(&e))
		sb.WriteString("\n")
	}

	for _, r := range feat.Routes {
		if r.Vouch != nil && r.Grabit != nil {
			model := findModel(proj, r.Grabit.Model)
			if model == nil {
				continue
			}
			if isCreateRoute(r) {
				sb.WriteString(genCreateRequest(r, model))
				sb.WriteString("\n")
			}
			if isUpdateRoute(r) {
				sb.WriteString(genUpdateRequest(r, model))
				sb.WriteString("\n")
			}
		}
		if r.Grabit != nil && len(r.Grabit.Wheres) > 0 {
			pascal := toPascalCase(r.Handler)
			sb.WriteString(fmt.Sprintf("type %sParams struct{\n", pascal))
			for _, w := range r.Grabit.Wheres {
				if w.Source.Kind == SourceQuery {
					sb.WriteString(fmt.Sprintf("\t%s *string\n", toPascalCase(w.Source.Value)))
				}
			}
			if r.Grabit.OrderBy != nil && r.Grabit.OrderBy.Kind == SourceQuery {
				sb.WriteString(fmt.Sprintf("\t%s string\n", toPascalCase(r.Grabit.OrderBy.Value)))
			}
			if r.Grabit.OrderDir != nil && r.Grabit.OrderDir.Kind == SourceQuery {
				sb.WriteString(fmt.Sprintf("\t%s string\n", toPascalCase(r.Grabit.OrderDir.Value)))
			}
			if r.Grabit.Limit != nil && r.Grabit.Limit.Kind == SourceQuery {
				sb.WriteString(fmt.Sprintf("\t%s int\n", toPascalCase(r.Grabit.Limit.Value)))
			}
			if r.Grabit.Offset != nil && r.Grabit.Offset.Kind == SourceQuery {
				sb.WriteString(fmt.Sprintf("\t%s int\n", toPascalCase(r.Grabit.Offset.Value)))
			}
			sb.WriteString("}\n\n")
		}
	}

	return sb.String()
}

func genModelStruct(m *ManifestDecl) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct{\n", m.Name))
	for _, f := range m.Fields {
		goType := comeTypeToGoNullable(f.Type, f.Nullable)
		tag := genJSONTag(f)
		sb.WriteString(fmt.Sprintf("\t%s %s %s\n", toPascalCase(f.Name), goType, tag))
	}
	sb.WriteString("}\n")
	return sb.String()
}

func genJSONTag(f FieldDecl) string {
	tag := f.Name
	if f.Nullable {
		tag += ",omitempty"
	}
	return fmt.Sprintf("`json:\"%s\"`", tag)
}

func genEnumConsts(e *EnumDecl) string {
	var sb strings.Builder
	sb.WriteString("const(\n")
	for _, v := range e.Values {
		sb.WriteString(fmt.Sprintf("\t%s%s=%q\n", e.Name, toPascalCase(v), v))
	}
	sb.WriteString(")\n\n")

	sb.WriteString(fmt.Sprintf("var Valid%s=map[string]bool{\n", e.Name))
	for _, v := range e.Values {
		sb.WriteString(fmt.Sprintf("\t%q:true,\n", v))
	}
	sb.WriteString("}\n")
	return sb.String()
}

func genCreateRequest(r RouteDecl, m *ManifestDecl) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	sb.WriteString(fmt.Sprintf("type %sRequest struct{\n", pascal))
	for _, vf := range r.Vouch.Fields {
		field := findFieldByName(m, vf.Name)
		if field == nil {
			continue
		}
		isOpt := hasDec(vf.Decorators, "optional")
		goType := comeTypeToGoNullable(field.Type, isOpt || field.Nullable)
		tag := fmt.Sprintf("`json:\"%s\"", vf.Name)
		if isOpt {
			tag += ",omitempty"
		}
		tag += "`"
		sb.WriteString(fmt.Sprintf("\t%s %s %s\n", toPascalCase(vf.Name), goType, tag))
	}
	sb.WriteString("}\n")
	return sb.String()
}

func genUpdateRequest(r RouteDecl, m *ManifestDecl) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	sb.WriteString(fmt.Sprintf("type %sRequest struct{\n", pascal))
	for _, vf := range r.Vouch.Fields {
		field := findFieldByName(m, vf.Name)
		if field == nil {
			continue
		}
		goType := comeTypeToGoNullable(field.Type, true)
		tag := fmt.Sprintf("`json:\"%s,omitempty\"`", vf.Name)
		sb.WriteString(fmt.Sprintf("\t%s %s %s\n", toPascalCase(vf.Name), goType, tag))
	}
	sb.WriteString("}\n")
	return sb.String()
}

func findFieldByName(m *ManifestDecl, name string) *FieldDecl {
	for i := range m.Fields {
		if m.Fields[i].Name == name {
			return &m.Fields[i]
		}
	}
	return nil
}
