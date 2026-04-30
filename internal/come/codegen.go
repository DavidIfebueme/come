package come

import (
	"fmt"
	"strings"
)

func Generate(proj *Project) map[string]string {
	files := map[string]string{}
	files["go.mod"] = GenGoMod(proj)
	files["cmd/server/main.go"] = GenMain(proj)
	files["pkg/database/pool.go"] = GenDatabase(proj)
	files["pkg/server/router.go"] = GenRouter(proj)
	files["pkg/server/middleware.go"] = GenMiddleware(proj)
	files["pkg/server/response.go"] = GenResponse(proj)
	if proj.Bouncer != nil {
		files["pkg/auth/jwt.go"] = GenAuth(proj)
	}
	for _, feat := range proj.Features {
		prefix := "internal/" + feat.Name + "/"
		files[prefix+"types.go"] = GenTypes(proj, feat)
		files[prefix+"handler.go"] = GenHandler(proj, feat)
		files[prefix+"repository.go"] = GenRepository(proj, feat)
		files[prefix+"routes.go"] = GenRoutes(proj, feat)
		for _, b := range feat.Babbles {
			files[prefix+"babble.go"] = GenBabble(proj, feat, b)
		}
	}
	for path, content := range GenMigrations(proj) {
		files[path] = content
	}
	if seedCode := GenSeed(proj); seedCode != "" {
		files["cmd/seed/main.go"] = seedCode
	}
	return files
}

func comeTypeToGo(ft FieldType) string {
	switch ft.Kind {
	case FieldString:
		return "string"
	case FieldInt:
		return "int"
	case FieldFloat:
		return "float64"
	case FieldBool:
		return "bool"
	case FieldTimestamp:
		return "time.Time"
	case FieldUUID:
		return "string"
	case FieldJSON:
		return "json.RawMessage"
	case FieldBytes:
		return "[]byte"
	case FieldEnum:
		return "string"
	case FieldArray:
		if ft.Items != nil {
			return "[]" + comeTypeToGo(*ft.Items)
		}
		return "[]any"
	default:
		return "any"
	}
}

func comeTypeToGoNullable(ft FieldType, nullable bool) string {
	base := comeTypeToGo(ft)
	if !nullable {
		return base
	}
	switch ft.Kind {
	case FieldString, FieldEnum, FieldUUID:
		return "*" + base
	case FieldInt:
		return "*int"
	case FieldFloat:
		return "*float64"
	case FieldBool:
		return "*bool"
	case FieldTimestamp:
		return "*" + base
	default:
		return base
	}
}

func comeTypeToSQL(ft FieldType, driver string) string {
	switch ft.Kind {
	case FieldString:
		return "TEXT"
	case FieldInt:
		return "INTEGER"
	case FieldFloat:
		return "REAL"
	case FieldBool:
		return "BOOLEAN"
	case FieldTimestamp:
		if driver == "postgres" {
			return "TIMESTAMPTZ"
		}
		return "DATETIME"
	case FieldUUID:
		if driver == "postgres" {
			return "UUID"
		}
		return "TEXT"
	case FieldJSON:
		if driver == "postgres" {
			return "JSONB"
		}
		return "TEXT"
	case FieldBytes:
		return "BLOB"
	case FieldEnum:
		return "TEXT"
	case FieldArray:
		if driver == "postgres" {
			if ft.Items != nil {
				return comeTypeToSQL(*ft.Items, driver) + "[]"
			}
			return "TEXT[]"
		}
		return "TEXT"
	default:
		return "TEXT"
	}
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func goPathPattern(comePath string) string {
	parts := strings.Split(comePath, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + p[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func hasDec(decs []Decorator, name string) bool {
	for _, d := range decs {
		if d.Name == name {
			return true
		}
	}
	return false
}

func decArg(decs []Decorator, name string) string {
	for _, d := range decs {
		if d.Name == name && len(d.Args) > 0 {
			return d.Args[0]
		}
	}
	return ""
}

func decArgs(decs []Decorator, name string) []string {
	for _, d := range decs {
		if d.Name == name {
			return d.Args
		}
	}
	return nil
}

func findModel(proj *Project, name string) *ManifestDecl {
	for i := range proj.Features {
		for j := range proj.Features[i].Manifests {
			if proj.Features[i].Manifests[j].Name == name {
				return &proj.Features[i].Manifests[j]
			}
		}
	}
	return nil
}

func findEnum(proj *Project, name string) *EnumDecl {
	for i := range proj.Features {
		for j := range proj.Features[i].Enums {
			if proj.Features[i].Enums[j].Name == name {
				return &proj.Features[i].Enums[j]
			}
		}
	}
	return nil
}

func primaryField(m *ManifestDecl) *FieldDecl {
	for i := range m.Fields {
		if hasDec(m.Fields[i].Decorators, "primary") {
			return &m.Fields[i]
		}
	}
	return nil
}

func modelColumns(m *ManifestDecl) string {
	cols := make([]string, len(m.Fields))
	for i, f := range m.Fields {
		cols[i] = toSnakeCase(f.Name)
	}
	return strings.Join(cols, ", ")
}

func modelScanFields(m *ManifestDecl, varName string) string {
	parts := make([]string, len(m.Fields))
	for i, f := range m.Fields {
		ft := f.Type
		if ft.Kind == FieldJSON {
			parts[i] = fmt.Sprintf("&%s.%s", varName, f.Name)
		} else if f.Nullable {
			parts[i] = fmt.Sprintf("&%s.%s", varName, f.Name)
		} else {
			parts[i] = fmt.Sprintf("&%s.%s", varName, f.Name)
		}
	}
	return strings.Join(parts, ", ")
}

func zeroValue(ft FieldType) string {
	switch ft.Kind {
	case FieldString, FieldEnum, FieldUUID:
		return `""`
	case FieldInt:
		return "0"
	case FieldFloat:
		return "0.0"
	case FieldBool:
		return "false"
	case FieldTimestamp:
		return "time.Time{}"
	case FieldJSON:
		return "nil"
	case FieldBytes:
		return "nil"
	case FieldArray:
		return "nil"
	default:
		return `""`
	}
}

func needsTimeImport(m *ManifestDecl) bool {
	for _, f := range m.Fields {
		if f.Type.Kind == FieldTimestamp {
			return true
		}
	}
	return false
}

func needsJSONImport(m *ManifestDecl) bool {
	for _, f := range m.Fields {
		if f.Type.Kind == FieldJSON {
			return true
		}
	}
	return false
}

func needsUUIDImport(m *ManifestDecl) bool {
	for _, f := range m.Fields {
		if f.Type.Kind == FieldUUID && hasDec(f.Decorators, "default") && decArg(f.Decorators, "default") == "gen_random_uuid" {
			return true
		}
	}
	return false
}

func featureHasTimestamp(feat Feature) bool {
	for _, m := range feat.Manifests {
		if needsTimeImport(&m) {
			return true
		}
	}
	return false
}

func featureHasJSON(feat Feature) bool {
	for _, m := range feat.Manifests {
		if needsJSONImport(&m) {
			return true
		}
	}
	return false
}

func featureHasUUID(feat Feature) bool {
	for _, m := range feat.Manifests {
		if needsUUIDImport(&m) {
			return true
		}
	}
	return false
}

func methodRepoName(route RouteDecl) string {
	return toPascalCase(route.Handler)
}

func tableName(manifestName string) string {
	return toSnakeCase(manifestName)
}

func isUpdateRoute(r RouteDecl) bool {
	return r.Method == "PUT" || r.Method == "PATCH"
}

func isCreateRoute(r RouteDecl) bool {
	return r.Method == "POST"
}

func isDeleteRoute(r RouteDecl) bool {
	return r.Method == "DELETE"
}

func isListRoute(r RouteDecl) bool {
	return r.Method == "GET" && r.Grabit != nil && !r.Grabit.One
}

func isGetRoute(r RouteDecl) bool {
	return r.Method == "GET" && r.Grabit != nil && r.Grabit.One
}
