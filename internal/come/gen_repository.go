package come

import (
	"fmt"
	"strings"
)

func GenRepository(proj *Project, feat Feature) string {
	var sb strings.Builder
	sb.WriteString("package " + feat.Name + "\n\nimport(\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"database/sql\"\n")
	sb.WriteString(fmt.Sprintf("\t\"%s/pkg/database\"\n", proj.AppName))

	needsFmt := false
	needsStrings := false
	needsTime := false
	needsStrconv := false
	for _, r := range feat.Routes {
		if r.Grabit != nil && r.Grabit.Operation == GrabitSelect && !r.Grabit.One {
			needsFmt = true
			needsStrings = true
		}
		if r.Grabit != nil && r.Grabit.Operation == GrabitUpdate {
			needsFmt = true
			needsStrings = true
			needsTime = true
		}
		if r.Grabit != nil && r.Grabit.Operation == GrabitInsert {
			needsTime = true
		}
		if r.Grabit != nil {
			for _, w := range r.Grabit.Wheres {
				if w.Source.Kind == SourceQuery && w.Op != "==" && w.Op != "!=" {
					model := findModel(proj, r.Grabit.Model)
					if model != nil {
						f := findFieldByName(model, w.Field)
						if f != nil && (f.Type.Kind == FieldInt || f.Type.Kind == FieldFloat) {
							needsStrconv = true
						}
					}
				}
			}
		}
	}
	if needsFmt {
		sb.WriteString("\t\"fmt\"\n")
	}
	if needsStrings {
		sb.WriteString("\t\"strings\"\n")
	}
	if needsTime {
		sb.WriteString("\t\"time\"\n")
	}
	if needsStrconv {
		sb.WriteString("\t\"strconv\"\n")
	}
	needsUUID := false
	for _, r := range feat.Routes {
		if r.Grabit != nil && r.Grabit.Operation == GrabitInsert {
			model := findModel(proj, r.Grabit.Model)
			if model != nil {
				for _, f := range model.Fields {
					if hasDec(f.Decorators, "primary") && hasDec(f.Decorators, "default") && decArg(f.Decorators, "default") == "gen_random_uuid" {
						needsUUID = true
					}
				}
			}
		}
	}
	if needsUUID {
		sb.WriteString("\t\"github.com/google/uuid\"\n")
	}
	sb.WriteString(")\n\n")

	sb.WriteString("type Repository struct{\n")
	sb.WriteString("\tdb *database.DB\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func NewRepository(db *database.DB)*Repository{\n")
	sb.WriteString("\treturn &Repository{db:db}\n")
	sb.WriteString("}\n\n")

	for _, m := range feat.Manifests {
		sb.WriteString(genScanHelper(&m))
	}

	for _, r := range feat.Routes {
		if r.Grabit == nil {
			continue
		}
		model := findModel(proj, r.Grabit.Model)
		if model == nil {
			continue
		}
		switch r.Grabit.Operation {
		case GrabitSelect:
			if r.Grabit.One {
				sb.WriteString(genGetByIDRepo(r, model, proj))
			} else {
				sb.WriteString(genListRepo(r, model, proj))
			}
		case GrabitInsert:
			sb.WriteString(genInsertRepo(r, model, proj))
		case GrabitUpdate:
			sb.WriteString(genUpdateRepo(r, model, proj))
		case GrabitDelete:
			sb.WriteString(genDeleteRepo(r, model, proj))
		}
	}

	return sb.String()
}

func genScanHelper(m *ManifestDecl) string {
	var sb strings.Builder
	pascal := m.Name
	sb.WriteString(fmt.Sprintf("func scan%s(row *sql.Row)(*%s,error){\n", pascal, pascal))
	sb.WriteString(fmt.Sprintf("\tvar m %s\n", pascal))
	sb.WriteString("\terr:=row.Scan(")
	for i, f := range m.Fields {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("&m.%s", toPascalCase(f.Name)))
	}
	sb.WriteString(")\n")
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\treturn nil,err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &m,nil\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("func scan%sRows(rows *sql.Rows)([]%s,error){\n", pascal, pascal))
	sb.WriteString(fmt.Sprintf("\tvar results []%s\n", pascal))
	sb.WriteString("\tfor rows.Next(){\n")
	sb.WriteString(fmt.Sprintf("\t\tvar m %s\n", pascal))
	sb.WriteString("\t\tif err:=rows.Scan(")
	for i, f := range m.Fields {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("&m.%s", toPascalCase(f.Name)))
	}
	sb.WriteString(");err!=nil{\n")
	sb.WriteString("\t\t\treturn nil,err\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tresults=append(results,m)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn results,rows.Err()\n")
	sb.WriteString("}\n\n")

	return sb.String()
}

func genGetByIDRepo(r RouteDecl, m *ManifestDecl, proj *Project) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	tbl := tableName(r.Grabit.Model)
	pk := primaryField(m)
	pkCol := "id"
	if pk != nil {
		pkCol = toSnakeCase(pk.Name)
	}

	sb.WriteString(fmt.Sprintf("func (r *Repository)%s(ctx context.Context,", pascal))
	paramParts := genPathParamParams(r)
	sb.WriteString(strings.Join(paramParts, ","))
	sb.WriteString(fmt.Sprintf(")(*%s,error){\n", m.Name))
	sb.WriteString(fmt.Sprintf("\tquery:=r.db.Rebind(\"SELECT %s FROM %s WHERE %s=$1\")\n", modelColumns(m), tbl, pkCol))
	sb.WriteString(fmt.Sprintf("\treturn scan%s(r.db.QueryRowContext(ctx,query,", m.Name))
	paramParts2 := genPathParamArgs(r)
	sb.WriteString(strings.Join(paramParts2, ","))
	sb.WriteString("))\n")
	sb.WriteString("}\n\n")
	return sb.String()
}

func genListRepo(r RouteDecl, m *ManifestDecl, proj *Project) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	tbl := tableName(r.Grabit.Model)

	sb.WriteString(fmt.Sprintf("func (r *Repository)%s(ctx context.Context,params %sParams)([]%s,int,error){\n", pascal, pascal, m.Name))
	sb.WriteString("\tvar conditions []string\n")
	sb.WriteString("\tvar args []any\n")
	sb.WriteString("\targPos:=1\n\n")

	for _, w := range r.Grabit.Wheres {
		colName := toSnakeCase(w.Field)
		switch w.Source.Kind {
		case SourceQuery:
			pascalSrc := toPascalCase(w.Source.Value)
			goOp := whereOpToGo(w.Op)
			sb.WriteString(fmt.Sprintf("\tif params.%s!=nil{\n", pascalSrc))
			if w.Op == "ilike" {
				sb.WriteString(fmt.Sprintf("\t\tconditions=append(conditions,fmt.Sprintf(\"%s ILIKE $%%d\",argPos))\n", colName))
				sb.WriteString(fmt.Sprintf("\t\targs=append(args,\"%%\"+*params.%s+\"%%\")\n", pascalSrc))
			} else {
				modelField := findFieldByName(m, w.Field)
				if modelField != nil && (modelField.Type.Kind == FieldInt || modelField.Type.Kind == FieldFloat) {
					if modelField.Type.Kind == FieldInt {
						sb.WriteString(fmt.Sprintf("\t\tif v,err:=strconv.Atoi(*params.%s);err==nil{\n", pascalSrc))
						sb.WriteString(fmt.Sprintf("\t\t\tconditions=append(conditions,fmt.Sprintf(\"%s %s $%%d\",argPos))\n", colName, goOp))
						sb.WriteString("\t\t\targs=append(args,v)\n")
						sb.WriteString("\t\t\targPos++\n")
						sb.WriteString("\t\t}\n")
					} else {
						sb.WriteString(fmt.Sprintf("\t\tif v,err:=strconv.ParseFloat(*params.%s,64);err==nil{\n", pascalSrc))
						sb.WriteString(fmt.Sprintf("\t\t\tconditions=append(conditions,fmt.Sprintf(\"%s %s $%%d\",argPos))\n", colName, goOp))
						sb.WriteString("\t\t\targs=append(args,v)\n")
						sb.WriteString("\t\t\targPos++\n")
						sb.WriteString("\t\t}\n")
					}
				} else {
					sb.WriteString(fmt.Sprintf("\t\tconditions=append(conditions,fmt.Sprintf(\"%s %s $%%d\",argPos))\n", colName, goOp))
					sb.WriteString(fmt.Sprintf("\t\targs=append(args,*params.%s)\n", pascalSrc))
					sb.WriteString("\t\targPos++\n")
				}
			}
			sb.WriteString("\t}\n")
		case SourceParam:
			sb.WriteString(fmt.Sprintf("\tconditions=append(conditions,fmt.Sprintf(\"%s %s $%%d\",argPos))\n", colName, whereOpToGo(w.Op)))
			sb.WriteString(fmt.Sprintf("\targs=append(args,%s)\n", toSnakeCase(w.Source.Value)))
			sb.WriteString("\targPos++\n")
		case SourceLiteral:
			sb.WriteString(fmt.Sprintf("\tconditions=append(conditions,\"%s %s %s\")\n", colName, whereOpToGo(w.Op), w.Source.Value))
		}
	}

	sb.WriteString("\n\twhereClause:=\"\"\n")
	sb.WriteString("\tif len(conditions)>0{\n")
	sb.WriteString("\t\twhereClause=\"WHERE \"+strings.Join(conditions,\" AND \")\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString(fmt.Sprintf("\tcountQuery:=r.db.Rebind(\"SELECT COUNT(*) FROM %s \"+whereClause)\n", tbl))
	sb.WriteString("\tvar total int\n")
	sb.WriteString("\tif err:=r.db.QueryRowContext(ctx,countQuery,args...).Scan(&total);err!=nil{\n")
	sb.WriteString("\t\treturn nil,0,err\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\torderBy:=\"created_at\"\n")
	if r.Grabit.OrderBy != nil && r.Grabit.OrderBy.Kind == SourceQuery {
		sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(r.Grabit.OrderBy.Value)))
		if len(r.Grabit.OrderByAllowed) > 0 {
			sb.WriteString(fmt.Sprintf("\t\tvalidSort:=map[string]bool{"))
			for _, val := range r.Grabit.OrderByAllowed {
				sb.WriteString(fmt.Sprintf("%q:true,", val))
			}
			sb.WriteString("}\n")
			sb.WriteString(fmt.Sprintf("\t\tif validSort[params.%s]{\n", toPascalCase(r.Grabit.OrderBy.Value)))
			sb.WriteString(fmt.Sprintf("\t\t\torderBy=params.%s\n", toPascalCase(r.Grabit.OrderBy.Value)))
			sb.WriteString("\t\t}\n")
		} else {
			sb.WriteString(fmt.Sprintf("\t\torderBy=params.%s\n", toPascalCase(r.Grabit.OrderBy.Value)))
		}
		sb.WriteString("\t}\n")
	}

	sb.WriteString("\torderDir:=\"DESC\"\n")
	if r.Grabit.OrderDir != nil && r.Grabit.OrderDir.Kind == SourceQuery {
		sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(r.Grabit.OrderDir.Value)))
		sb.WriteString(fmt.Sprintf("\t\torderDir=strings.ToUpper(params.%s)\n", toPascalCase(r.Grabit.OrderDir.Value)))
		sb.WriteString("\t}\n")
	}

	sb.WriteString("\tlimit:=20\n")
	if r.Grabit.Limit != nil && r.Grabit.Limit.Kind == SourceQuery {
		sb.WriteString(fmt.Sprintf("\tif params.%s>0{\n", toPascalCase(r.Grabit.Limit.Value)))
		sb.WriteString(fmt.Sprintf("\t\tlimit=params.%s\n", toPascalCase(r.Grabit.Limit.Value)))
		sb.WriteString("\t}\n")
	}

	if r.Grabit.Page != nil && r.Grabit.Page.Kind == SourceQuery {
		sb.WriteString("\tpage:=1\n")
		sb.WriteString(fmt.Sprintf("\tif params.%s>0{\n", toPascalCase(r.Grabit.Page.Value)))
		sb.WriteString(fmt.Sprintf("\t\tpage=params.%s\n", toPascalCase(r.Grabit.Page.Value)))
		sb.WriteString("\t}\n")
		sb.WriteString("\toffset:=(page-1)*limit\n")
	} else if r.Grabit.Offset != nil && r.Grabit.Offset.Kind == SourceQuery {
		sb.WriteString("\toffset:=0\n")
		sb.WriteString(fmt.Sprintf("\tif params.%s>0{\n", toPascalCase(r.Grabit.Offset.Value)))
		sb.WriteString(fmt.Sprintf("\t\toffset=params.%s\n", toPascalCase(r.Grabit.Offset.Value)))
		sb.WriteString("\t}\n")
	} else {
		sb.WriteString("\toffset:=0\n")
	}

	sb.WriteString(fmt.Sprintf("\tquery:=r.db.Rebind(fmt.Sprintf(\"SELECT %s FROM %s \"+whereClause+\" ORDER BY \"+orderBy+\" \"+orderDir+\" LIMIT $%%d OFFSET $%%d\",argPos,argPos+1))\n", modelColumns(m), tbl))
	sb.WriteString("\targs=append(args,limit,offset)\n\n")

	sb.WriteString(fmt.Sprintf("\trows,err:=r.db.QueryContext(ctx,query,args...)\n"))
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\treturn nil,0,err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tdefer rows.Close()\n\n")

	sb.WriteString(fmt.Sprintf("\tresults,err:=scan%sRows(rows)\n", m.Name))
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\treturn nil,0,err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn results,total,nil\n")
	sb.WriteString("}\n\n")

	return sb.String()
}

func genInsertRepo(r RouteDecl, m *ManifestDecl, proj *Project) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	tbl := tableName(r.Grabit.Model)

	vouchFields := map[string]bool{}
	if r.Vouch != nil {
		for _, vf := range r.Vouch.Fields {
			vouchFields[vf.Name] = true
		}
	}

	var insertCols []string
	var insertArgs []string
	var extraArgExprs []string
	n := 1
	for _, f := range m.Fields {
		if hasDec(f.Decorators, "primary") && hasDec(f.Decorators, "default") {
			if decArg(f.Decorators, "default") == "gen_random_uuid" {
				insertCols = append(insertCols, toSnakeCase(f.Name))
				insertArgs = append(insertArgs, fmt.Sprintf("$%d", n))
				extraArgExprs = append(extraArgExprs, "uuid.Must(uuid.NewV7()).String()")
				n++
				continue
			}
		}
		if hasDec(f.Decorators, "auto_update") {
			continue
		}
		if f.Type.Kind == FieldTimestamp && hasDec(f.Decorators, "default") && decArg(f.Decorators, "default") == "now" {
			insertCols = append(insertCols, toSnakeCase(f.Name))
			insertArgs = append(insertArgs, fmt.Sprintf("$%d", n))
			extraArgExprs = append(extraArgExprs, "time.Now()")
			n++
			continue
		}
		if len(vouchFields) > 0 && !vouchFields[f.Name] {
			continue
		}
		insertCols = append(insertCols, toSnakeCase(f.Name))
		insertArgs = append(insertArgs, fmt.Sprintf("$%d", n))
		n++
	}

	sb.WriteString(fmt.Sprintf("func (r *Repository)%s(ctx context.Context,req %sRequest)(*%s,error){\n", pascal, pascal, m.Name))
	sb.WriteString(fmt.Sprintf("\tquery:=r.db.Rebind(\"INSERT INTO %s (%s) VALUES (%s) RETURNING %s\")\n",
		tbl, strings.Join(insertCols, ", "), strings.Join(insertArgs, ","), modelColumns(m)))

	var argParts []string
	argIdx := 0
	for _, f := range m.Fields {
		if hasDec(f.Decorators, "primary") && hasDec(f.Decorators, "default") && decArg(f.Decorators, "default") == "gen_random_uuid" {
			argParts = append(argParts, extraArgExprs[argIdx])
			argIdx++
			continue
		}
		if hasDec(f.Decorators, "auto_update") {
			continue
		}
		if f.Type.Kind == FieldTimestamp && hasDec(f.Decorators, "default") && decArg(f.Decorators, "default") == "now" {
			argParts = append(argParts, extraArgExprs[argIdx])
			argIdx++
			continue
		}
		if len(vouchFields) > 0 && !vouchFields[f.Name] {
			continue
		}
		argParts = append(argParts, fmt.Sprintf("req.%s", toPascalCase(f.Name)))
	}
	sb.WriteString(fmt.Sprintf("\targs:=[]any{%s}\n\n", strings.Join(argParts, ",")))

	sb.WriteString(fmt.Sprintf("\treturn scan%s(r.db.QueryRowContext(ctx,query,args...))\n", m.Name))
	sb.WriteString("}\n\n")

	return sb.String()
}

func genUpdateRepo(r RouteDecl, m *ManifestDecl, proj *Project) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	tbl := tableName(r.Grabit.Model)

	sb.WriteString(fmt.Sprintf("func (r *Repository)%s(ctx context.Context,", pascal))
	paramParts := genPathParamParams(r)
	sb.WriteString(strings.Join(paramParts, ","))
	if len(paramParts) > 0 {
		sb.WriteString(",")
	}
	sb.WriteString(fmt.Sprintf("req %sRequest)(*%s,error){\n", pascal, m.Name))

	sb.WriteString("\tvar sets []string\n")
	sb.WriteString("\tvar args []any\n")
	sb.WriteString("\targPos:=1\n\n")

	for _, f := range m.Fields {
		if hasDec(f.Decorators, "primary") {
			continue
		}
		if hasDec(f.Decorators, "auto_update") {
			continue
		}
		if f.Type.Kind == FieldTimestamp && hasDec(f.Decorators, "auto_update") {
			continue
		}
		if r.Vouch != nil {
			hasField := false
			for _, vf := range r.Vouch.Fields {
				if vf.Name == f.Name {
					hasField = true
					break
				}
			}
			if !hasField {
				continue
			}
		}
		sb.WriteString(fmt.Sprintf("\tif req.%s!=nil{\n", toPascalCase(f.Name)))
		sb.WriteString(fmt.Sprintf("\t\tsets=append(sets,fmt.Sprintf(\"%s=$%%d\",argPos))\n", toSnakeCase(f.Name)))
		sb.WriteString(fmt.Sprintf("\t\targs=append(args,*req.%s)\n", toPascalCase(f.Name)))
		sb.WriteString("\t\targPos++\n")
		sb.WriteString("\t}\n")
	}

	var autoUpdateField *FieldDecl
	for i := range m.Fields {
		if hasDec(m.Fields[i].Decorators, "auto_update") {
			autoUpdateField = &m.Fields[i]
			break
		}
	}
	if autoUpdateField != nil {
		sb.WriteString(fmt.Sprintf("\tsets=append(sets,fmt.Sprintf(\"%s=$%%d\",argPos))\n", toSnakeCase(autoUpdateField.Name)))
		sb.WriteString("\targs=append(args,time.Now())\n")
		sb.WriteString("\targPos++\n\n")
	}

	pk := primaryField(m)
	sb.WriteString("\tif len(sets)==0{\n")
	if pk != nil {
		sb.WriteString(fmt.Sprintf("\t\tvar zero %s\n", m.Name))
		sb.WriteString("\t\treturn &zero,fmt.Errorf(\"no fields to update\")\n")
	} else {
		sb.WriteString("\t\treturn nil,fmt.Errorf(\"no fields to update\")\n")
	}
	sb.WriteString("\t}\n\n")

	pathArgs := genPathParamArgs(r)
	sb.WriteString("\targs=append(args," + strings.Join(pathArgs, ",") + ")\n\n")

	pkCol := "id"
	if pk != nil {
		pkCol = toSnakeCase(pk.Name)
	}
	sb.WriteString(fmt.Sprintf("\tquery:=r.db.Rebind(fmt.Sprintf(\"UPDATE %s SET \"+strings.Join(sets,\", \")+\" WHERE %s=$%%d RETURNING %s\",argPos))\n",
		tbl, pkCol, modelColumns(m)))

	sb.WriteString(fmt.Sprintf("\treturn scan%s(r.db.QueryRowContext(ctx,query,args...))\n", m.Name))
	sb.WriteString("}\n\n")

	return sb.String()
}

func genDeleteRepo(r RouteDecl, m *ManifestDecl, proj *Project) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	tbl := tableName(r.Grabit.Model)
	pk := primaryField(m)
	pkCol := "id"
	if pk != nil {
		pkCol = toSnakeCase(pk.Name)
	}

	sb.WriteString(fmt.Sprintf("func (r *Repository)%s(ctx context.Context,", pascal))
	paramParts := genPathParamParams(r)
	sb.WriteString(strings.Join(paramParts, ","))
	sb.WriteString(")error{\n")
	sb.WriteString(fmt.Sprintf("\tquery:=r.db.Rebind(\"DELETE FROM %s WHERE %s=$1\")\n", tbl, pkCol))
	sb.WriteString(fmt.Sprintf("\t_,err:=r.db.ExecContext(ctx,query,"))
	paramArgs := genPathParamArgs(r)
	sb.WriteString(strings.Join(paramArgs, ","))
	sb.WriteString(")\n")
	sb.WriteString("\treturn err\n")
	sb.WriteString("}\n\n")

	return sb.String()
}

func whereOpToGo(op string) string {
	switch op {
	case "==":
		return "="
	case "!=":
		return "!="
	case ">=":
		return ">="
	case "<=":
		return "<="
	case ">":
		return ">"
	case "<":
		return "<"
	case "ilike":
		return "ILIKE"
	default:
		return "="
	}
}

func genPathParamParams(r RouteDecl) []string {
	var params []string
	for _, w := range r.Grabit.Wheres {
		if w.Source.Kind == SourceParam {
			params = append(params, fmt.Sprintf("%s string", toSnakeCase(w.Source.Value)))
		}
	}
	return params
}

func genPathParamArgs(r RouteDecl) []string {
	var args []string
	for _, w := range r.Grabit.Wheres {
		if w.Source.Kind == SourceParam {
			args = append(args, toSnakeCase(w.Source.Value))
		}
	}
	return args
}

func pathVarArgName(pk *FieldDecl) string {
	if pk == nil {
		return "id"
	}
	return toSnakeCase(pk.Name)
}

func firstGetterName(feat Feature, m *ManifestDecl) string {
	return "Get" + m.Name + "ByID"
}
