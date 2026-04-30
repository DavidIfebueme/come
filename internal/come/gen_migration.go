package come

import (
	"fmt"
	"strings"
)

func GenMigrations(proj *Project) map[string]string {
	files := map[string]string{}
	seq := 1
	for _, feat := range proj.Features {
		for _, m := range feat.Manifests {
			up := genMigrationUp(&m, proj.Driver)
			down := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName(m.Name))
			num := fmt.Sprintf("%06d", seq)
			name := fmt.Sprintf("create_%s", tableName(m.Name))
			files[fmt.Sprintf("migrations/%s_%s.up.sql", num, name)] = up
			files[fmt.Sprintf("migrations/%s_%s.down.sql", num, name)] = down
			seq++
		}
	}
	return files
}

func genMigrationUp(m *ManifestDecl, driver string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", tableName(m.Name)))
	cols := make([]string, 0, len(m.Fields))
	for _, f := range m.Fields {
		col := genColumnDef(f, driver)
		cols = append(cols, col)
	}
	sb.WriteString(strings.Join(cols, ",\n"))
	sb.WriteString("\n);\n")

	for _, idx := range m.Indexes {
		sb.WriteString(genIndexDef(m, idx, driver))
	}

	return sb.String()
}

func genColumnDef(f FieldDecl, driver string) string {
	sqlType := comeTypeToSQL(f.Type, driver)
	col := fmt.Sprintf("  %s %s", toSnakeCase(f.Name), sqlType)

	if hasDec(f.Decorators, "primary") {
		if f.Type.Kind == FieldUUID && driver == "postgres" {
			col += " PRIMARY KEY DEFAULT gen_random_uuid()"
		} else if f.Type.Kind == FieldUUID {
			col += " PRIMARY KEY"
		} else {
			col += " PRIMARY KEY"
			if def := decArg(f.Decorators, "default"); def != "" {
				col += fmt.Sprintf(" DEFAULT %s", defaultToSQL(def, f.Type, driver))
			}
		}
	} else {
		if !f.Nullable && !hasDec(f.Decorators, "primary") {
			if !hasDec(f.Decorators, "default") && !hasDec(f.Decorators, "optional") {
				col += " NOT NULL"
			}
		}
		if def := decArg(f.Decorators, "default"); def != "" {
			col += fmt.Sprintf(" DEFAULT %s", defaultToSQL(def, f.Type, driver))
		}
		if hasDec(f.Decorators, "unique") {
			col += " UNIQUE"
		}
	}

	if hasDec(f.Decorators, "homie") {
		ref := decArg(f.Decorators, "homie")
		if ref != "" {
			col += fmt.Sprintf(" REFERENCES %s(id)", tableName(ref))
		}
	}

	return col
}

func defaultToSQL(def string, ft FieldType, driver string) string {
	switch def {
	case "now":
		if driver == "postgres" {
			return "NOW()"
		}
		return "CURRENT_TIMESTAMP"
	case "gen_random_uuid":
		if driver == "postgres" {
			return "gen_random_uuid()"
		}
		return "(lower(hex(randomblob(16))))"
	default:
		if ft.Kind == FieldString || ft.Kind == FieldEnum {
			return fmt.Sprintf("'%s'", def)
		}
		return def
	}
}

func genIndexDef(m *ManifestDecl, idx IndexDecl, driver string) string {
	cols := make([]string, len(idx.Fields))
	for i, f := range idx.Fields {
		cols[i] = toSnakeCase(f)
	}
	idxName := fmt.Sprintf("idx_%s_%s", tableName(m.Name), strings.Join(cols, "_"))
	unique := ""
	if idx.Unique {
		unique = "UNIQUE "
	}
	return fmt.Sprintf("\nCREATE %sINDEX IF NOT EXISTS %s ON %s (%s);", unique, idxName, tableName(m.Name), strings.Join(cols, ", "))
}
