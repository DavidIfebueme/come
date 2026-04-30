package come

import (
	"fmt"
	"strings"
)

func GenSeed(proj *Project) string {
	var sb strings.Builder
	sb.WriteString("package main\n\nimport(\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"log\"\n")
	sb.WriteString("\t\"os\"\n")
	sb.WriteString("\t\"time\"\n\n")
	sb.WriteString(fmt.Sprintf("\t\"%s/pkg/database\"\n", proj.AppName))
	sb.WriteString("\t\"github.com/google/uuid\"\n")
	sb.WriteString(")\n\n")

	hasSeeds := false
	for _, feat := range proj.Features {
		if len(feat.Seeds) > 0 {
			hasSeeds = true
			break
		}
	}
	if !hasSeeds {
		return ""
	}

	sb.WriteString("func main(){\n")
	sb.WriteString("\tdbURL:=os.Getenv(\"DATABASE_URL\")\n")
	if len(proj.DBs) > 0 {
		sb.WriteString("\tif dbURL==\"\"{\n")
		sb.WriteString(fmt.Sprintf("\t\tdbURL=%q\n", proj.DBs[0].Connection))
		sb.WriteString("\t}\n")
	}
	sb.WriteString("\tdb,cleanup,err:=database.Connect(dbURL)\n")
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\tlog.Fatal(err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tdefer cleanup()\n\n")

	for _, feat := range proj.Features {
		for _, seed := range feat.Seeds {
			model := findModel(proj, feat.Manifests[0].Name)
			if model == nil {
				continue
			}
			sb.WriteString(genSeedBlock(seed, model, proj))
		}
	}

	sb.WriteString("\tfmt.Println(\"seeding complete\")\n")
	sb.WriteString("}\n")

	return sb.String()
}

func genSeedBlock(seed SpawnChaosDecl, model *ManifestDecl, proj *Project) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\tdata,err:=os.ReadFile(%q)\n", seed.Root))
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\tlog.Fatal(err)\n")
	sb.WriteString("\t}\n\n")

	if seed.Key != "" {
		sb.WriteString(fmt.Sprintf("\tvar wrapper struct{%s []json.RawMessage `json:%q`}\n", toPascalCase(seed.Key), seed.Key))
		sb.WriteString("\tif err:=json.Unmarshal(data,&wrapper);err!=nil{\n")
		sb.WriteString("\t\tlog.Fatal(err)\n")
		sb.WriteString("\t}\n")
		sb.WriteString(fmt.Sprintf("\titems:=wrapper.%s\n", toPascalCase(seed.Key)))
	} else {
		sb.WriteString("\tvar items []json.RawMessage\n")
		sb.WriteString("\tif err:=json.Unmarshal(data,&items);err!=nil{\n")
		sb.WriteString("\t\tlog.Fatal(err)\n")
		sb.WriteString("\t}\n")
	}

	sb.WriteString(fmt.Sprintf("\tfor _,item:=range items{\n"))
	sb.WriteString(fmt.Sprintf("\t\tvar m struct{\n"))

	for _, f := range model.Fields {
		if hasDec(f.Decorators, "primary") && hasDec(f.Decorators, "default") {
			continue
		}
		if f.Type.Kind == FieldTimestamp && hasDec(f.Decorators, "default") && decArg(f.Decorators, "default") == "now" {
			continue
		}
		if hasDec(f.Decorators, "auto_update") {
			continue
		}
		goType := comeTypeToGo(f.Type)
		sb.WriteString(fmt.Sprintf("\t\t\t%s %s `json:%q`\n", toPascalCase(f.Name), goType, f.Name))
	}
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tif err:=json.Unmarshal(item,&m);err!=nil{\n")
	sb.WriteString("\t\t\tlog.Fatal(err)\n")
	sb.WriteString("\t\t}\n\n")

	var insertCols []string
	var insertPlaceholders []string
	var insertArgs []string
	n := 1
	for _, f := range model.Fields {
		if hasDec(f.Decorators, "primary") && hasDec(f.Decorators, "default") {
			if f.Type.Kind == FieldUUID {
				insertCols = append(insertCols, toSnakeCase(f.Name))
				insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", n))
				insertArgs = append(insertArgs, "uuid.Must(uuid.NewV7()).String()")
				n++
				continue
			}
			continue
		}
		if f.Type.Kind == FieldTimestamp && hasDec(f.Decorators, "default") && decArg(f.Decorators, "default") == "now" {
			insertCols = append(insertCols, toSnakeCase(f.Name))
			insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", n))
			insertArgs = append(insertArgs, "time.Now()")
			n++
			continue
		}
		if hasDec(f.Decorators, "auto_update") {
			insertCols = append(insertCols, toSnakeCase(f.Name))
			insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", n))
			insertArgs = append(insertArgs, "time.Now()")
			n++
			continue
		}
		insertCols = append(insertCols, toSnakeCase(f.Name))
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", n))
		insertArgs = append(insertArgs, fmt.Sprintf("m.%s", toPascalCase(f.Name)))
		n++
	}

	sb.WriteString(fmt.Sprintf("\t\tquery:=db.Rebind(\"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO NOTHING\")\n",
		tableName(model.Name),
		strings.Join(insertCols, ", "),
		strings.Join(insertPlaceholders, ","),
		toSnakeCase(seed.Unique)))
	sb.WriteString(fmt.Sprintf("\t\targs:=[]any{%s}\n", strings.Join(insertArgs, ",")))
	sb.WriteString("\t\tif _,err:=db.Exec(query,args...);err!=nil{\n")
	sb.WriteString("\t\t\tlog.Fatal(err)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n\n")

	return sb.String()
}
