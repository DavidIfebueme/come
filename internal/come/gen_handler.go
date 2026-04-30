package come

import (
	"fmt"
	"strings"
)

func GenHandler(proj *Project, feat Feature) string {
	var sb strings.Builder
	sb.WriteString("package " + feat.Name + "\n\nimport(\n")

	needsSQL := false
	for _, r := range feat.Routes {
		for _, hurl := range r.Hurls {
			if hurl.StatusCode == 404 && r.Grabit != nil && (r.Grabit.One || r.Grabit.Operation == GrabitUpdate) {
				needsSQL = true
			}
		}
	}
	if needsSQL {
		sb.WriteString("\t\"database/sql\"\n")
	}
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString(fmt.Sprintf("\t\"%s/pkg/server\"\n", proj.AppName))
	if proj.Bouncer != nil {
		sb.WriteString(fmt.Sprintf("\t\"%s/pkg/auth\"\n", proj.AppName))
	}

	needsStrconv := false
	for _, r := range feat.Routes {
		if r.Grabit != nil {
			if r.Grabit.Limit != nil || r.Grabit.Offset != nil {
				needsStrconv = true
			}
		}
	}
	if needsStrconv {
		sb.WriteString("\t\"strconv\"\n")
	}

	needsStrings := false
	for _, r := range feat.Routes {
		if r.Vouch != nil {
			for _, vf := range r.Vouch.Fields {
				if hasDec(vf.Decorators, "email") {
					needsStrings = true
				}
			}
		}
	}
	if needsStrings {
		sb.WriteString("\t\"strings\"\n")
	}

	sb.WriteString(")\n\n")

	sb.WriteString("type Handler struct{\n")
	sb.WriteString("\trepo *Repository\n")
	if proj.Bouncer != nil {
		sb.WriteString("\tauth *auth.JWTAuth\n")
	}
	sb.WriteString("}\n\n")

	sb.WriteString("func NewHandler(repo *Repository")
	if proj.Bouncer != nil {
		sb.WriteString(",auth *auth.JWTAuth")
	}
	sb.WriteString(")*Handler{\n")
	sb.WriteString("\treturn &Handler{repo:repo")
	if proj.Bouncer != nil {
		sb.WriteString(",auth:auth")
	}
	sb.WriteString("}\n")
	sb.WriteString("}\n\n")

	for _, r := range feat.Routes {
		sb.WriteString(genHandlerMethod(proj, feat, r))
	}

	for _, r := range feat.Routes {
		if r.Vouch != nil {
			sb.WriteString(genValidationFunc(proj, r))
		}
		if isListRoute(r) && r.Grabit != nil {
			sb.WriteString(genParamParser(r))
		}
	}

	hasEmail := false
	for _, r := range feat.Routes {
		if r.Vouch != nil {
			for _, vf := range r.Vouch.Fields {
				if hasDec(vf.Decorators, "email") {
					hasEmail = true
				}
			}
		}
	}
	if hasEmail {
		sb.WriteString("func isValidEmail(s string)bool{\n")
		sb.WriteString("\treturn strings.Contains(s,\"@\")&&strings.Contains(s,\".\")\n")
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func genHandlerMethod(proj *Project, feat Feature, r RouteDecl) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)

	sb.WriteString(fmt.Sprintf("func (h *Handler)%s(w http.ResponseWriter,r *http.Request){\n", pascal))

	for _, w := range r.Wards {
		if w.Name == "bouncer" {
			sb.WriteString("\tif h.auth!=nil{\n")
			if len(w.Args) > 0 {
				sb.WriteString("\t\tclaims,err:=h.auth.Validate(r)\n")
			} else {
				sb.WriteString("\t\t_,err:=h.auth.Validate(r)\n")
			}
			sb.WriteString("\t\tif err!=nil{\n")
			sb.WriteString("\t\t\tserver.Unauthorized(w)\n")
			sb.WriteString("\t\t\treturn\n")
			sb.WriteString("\t\t}\n")
			if len(w.Args) > 0 {
				sb.WriteString(fmt.Sprintf("\t\tif claims.Role!=%q{\n", w.Args[0]))
				sb.WriteString("\t\t\tserver.Forbidden(w)\n")
				sb.WriteString("\t\t\treturn\n")
				sb.WriteString("\t\t}\n")
			}
			sb.WriteString("\t}\n")
		}
	}

	if isCreateRoute(r) || isUpdateRoute(r) {
		sb.WriteString(fmt.Sprintf("\tvar req %sRequest\n", pascal))
		sb.WriteString("\tif err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{\n")
		sb.WriteString("\t\tserver.Error(w,http.StatusBadRequest,\"invalid request body\")\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n")
		if r.Vouch != nil {
			sb.WriteString(fmt.Sprintf("\tif errs:=validate%s(req);len(errs)>0{\n", pascal))
			sb.WriteString("\t\tserver.ValidationErrors(w,errs)\n")
			sb.WriteString("\t\treturn\n")
			sb.WriteString("\t}\n")
		}
	}

	if isListRoute(r) && r.Grabit != nil {
		sb.WriteString(fmt.Sprintf("\tparams:=parse%sParams(r)\n", pascal))
	}

	if isGetRoute(r) || isDeleteRoute(r) {
		for _, w := range r.Grabit.Wheres {
			if w.Source.Kind == SourceParam {
				sb.WriteString(fmt.Sprintf("\t%s:=r.PathValue(%q)\n", toSnakeCase(w.Source.Value), w.Source.Value))
			}
		}
	}

	if isUpdateRoute(r) {
		for _, w := range r.Grabit.Wheres {
			if w.Source.Kind == SourceParam {
				sb.WriteString(fmt.Sprintf("\t%s:=r.PathValue(%q)\n", toSnakeCase(w.Source.Value), w.Source.Value))
			}
		}
	}

	hasNotFound := false
	notFoundMsg := "not found"
	for _, hurl := range r.Hurls {
		if hurl.StatusCode == 404 {
			hasNotFound = true
			if hurl.Kind == HurlCustom {
				if msg, ok := hurl.CustomObj["message"]; ok {
					notFoundMsg = msg
				}
			}
		}
	}

	if r.Grabit != nil {
		switch r.Grabit.Operation {
		case GrabitSelect:
			if r.Grabit.One {
				sb.WriteString(fmt.Sprintf("\tresult,err:=h.repo.%s(r.Context(),", pascal))
				params := genPathParamArgs(r)
				sb.WriteString(strings.Join(params, ","))
				sb.WriteString(")\n")
			} else {
				sb.WriteString(fmt.Sprintf("\tresults,total,err:=h.repo.%s(r.Context(),params)\n", pascal))
			}
		case GrabitInsert:
			sb.WriteString(fmt.Sprintf("\tresult,err:=h.repo.%s(r.Context(),req)\n", pascal))
		case GrabitUpdate:
			sb.WriteString(fmt.Sprintf("\tresult,err:=h.repo.%s(r.Context(),", pascal))
			params := genPathParamArgs(r)
			sb.WriteString(strings.Join(params, ","))
			sb.WriteString(",req)\n")
		case GrabitDelete:
			sb.WriteString(fmt.Sprintf("\terr:=h.repo.%s(r.Context(),", pascal))
			params := genPathParamArgs(r)
			sb.WriteString(strings.Join(params, ","))
			sb.WriteString(")\n")
		}

		sb.WriteString("\tif err!=nil{\n")
		if hasNotFound && (r.Grabit.One || r.Grabit.Operation == GrabitUpdate) {
			sb.WriteString("\t\tif err==sql.ErrNoRows{\n")
			sb.WriteString(fmt.Sprintf("\t\t\tserver.Error(w,http.StatusNotFound,%q)\n", notFoundMsg))
			sb.WriteString("\t\t\treturn\n")
			sb.WriteString("\t\t}\n")
		}
		sb.WriteString("\t\tserver.Error(w,http.StatusInternalServerError,\"internal server error\")\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n")
	}

	if r.Bouncer != nil {
		for _, action := range r.Bouncer.Actions {
			if action.Kind == BouncerSign {
				sb.WriteString("\ttoken,err:=h.auth.Sign(")
				if sub, ok := action.Fields["sub"]; ok {
					sb.WriteString(sourceToHandlerArg(sub))
				} else {
					sb.WriteString("\"\"")
				}
				sb.WriteString(",")
				if role, ok := action.Fields["role"]; ok {
					sb.WriteString(sourceToHandlerArg(role))
				} else {
					sb.WriteString("\"\"")
				}
				sb.WriteString(")\n")
				sb.WriteString("\tif err!=nil{\n")
				sb.WriteString("\t\tserver.Error(w,http.StatusInternalServerError,\"failed to generate token\")\n")
				sb.WriteString("\t\treturn\n")
				sb.WriteString("\t}\n")
				sb.WriteString("\t_ = token\n")
			}
		}
	}

	for _, hurl := range r.Hurls {
		if hurl.StatusCode >= 200 && hurl.StatusCode < 300 {
			switch hurl.Kind {
			case HurlNoContent:
				sb.WriteString(fmt.Sprintf("\tw.WriteHeader(%d)\n", hurl.StatusCode))
			case HurlResult:
				sb.WriteString("\tw.Header().Set(\"Content-Type\",\"application/json\")\n")
				if hurl.StatusCode != 200 {
					sb.WriteString(fmt.Sprintf("\tw.WriteHeader(%d)\n", hurl.StatusCode))
				}
				if r.Grabit != nil && !r.Grabit.One && r.Grabit.Operation == GrabitSelect {
					sb.WriteString("\tjson.NewEncoder(w).Encode(map[string]any{\"data\":results,\"total\":total,\"page\":(params.Offset/params.Limit)+1,\"limit\":params.Limit})\n")
				} else {
					sb.WriteString("\tjson.NewEncoder(w).Encode(result)\n")
				}
			case HurlCustom:
				sb.WriteString("\tw.Header().Set(\"Content-Type\",\"application/json\")\n")
				if hurl.StatusCode != 200 {
					sb.WriteString(fmt.Sprintf("\tw.WriteHeader(%d)\n", hurl.StatusCode))
				}
				sb.WriteString("\tjson.NewEncoder(w).Encode(map[string]any{")
				first := true
				for k, v := range hurl.CustomObj {
					if !first {
						sb.WriteString(",")
					}
					first = false
					sb.WriteString(fmt.Sprintf("%q:%q", k, v))
				}
				sb.WriteString("})\n")
			}
			break
		}
	}

	sb.WriteString("}\n\n")
	return sb.String()
}

func sourceToHandlerArg(src ValueSource) string {
	switch src.Kind {
	case SourceResult:
		return fmt.Sprintf("result.%s", toPascalCase(src.Value))
	case SourceBody:
		return fmt.Sprintf("req.%s", toPascalCase(src.Value))
	case SourceLiteral:
		return fmt.Sprintf("%q", src.Value)
	default:
		return fmt.Sprintf("%q", src.Value)
	}
}

func genValidationFunc(proj *Project, r RouteDecl) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)
	model := findModel(proj, r.Grabit.Model)
	if model == nil {
		return ""
	}
	isUpdate := isUpdateRoute(r)

	sb.WriteString(fmt.Sprintf("func validate%s(req %sRequest)[]string{\n", pascal, pascal))
	sb.WriteString("\tvar errs []string\n")

	for _, vf := range r.Vouch.Fields {
		field := findFieldByName(model, vf.Name)
		if field == nil {
			continue
		}
		isOpt := hasDec(vf.Decorators, "optional")
		pname := toPascalCase(vf.Name)

		switch field.Type.Kind {
		case FieldString, FieldEnum, FieldUUID:
			if isUpdate {
				sb.WriteString(fmt.Sprintf("\tif req.%s!=nil{\n", pname))
				sb.WriteString(fmt.Sprintf("\t\tval:=*req.%s\n", pname))
				genStringValidation(&sb, "val", vf, field)
				sb.WriteString("\t}\n")
			} else {
				if !isOpt && hasDec(vf.Decorators, "required") {
					sb.WriteString(fmt.Sprintf("\tif req.%s==\"\"{\n", pname))
					sb.WriteString(fmt.Sprintf("\t\terrs=append(errs,%q)\n", vf.Name+" is required"))
					sb.WriteString("\t}\n")
				}
				sb.WriteString(fmt.Sprintf("\tif req.%s!=\"\"{\n", pname))
				genStringValidation(&sb, "req."+pname, vf, field)
				sb.WriteString("\t}\n")
			}
		case FieldInt, FieldFloat:
			if isUpdate {
				sb.WriteString(fmt.Sprintf("\tif req.%s!=nil{\n", pname))
				sb.WriteString(fmt.Sprintf("\t\tval:=*req.%s\n", pname))
				genNumericValidation(&sb, "val", vf)
				sb.WriteString("\t}\n")
			} else {
				if !isOpt && hasDec(vf.Decorators, "required") {
					sb.WriteString(fmt.Sprintf("\tif req.%s==nil{\n", pname))
					sb.WriteString(fmt.Sprintf("\t\terrs=append(errs,%q)\n", vf.Name+" is required"))
					sb.WriteString("\t}\n")
				}
				sb.WriteString(fmt.Sprintf("\tif req.%s!=nil{\n", pname))
				genNumericValidation(&sb, "*req."+pname, vf)
				sb.WriteString("\t}\n")
			}
		}
	}

	sb.WriteString("\treturn errs\n")
	sb.WriteString("}\n\n")
	return sb.String()
}

func genStringValidation(sb *strings.Builder, varName string, vf VouchField, field *FieldDecl) {
	if hasDec(vf.Decorators, "email") {
		sb.WriteString(fmt.Sprintf("\t\tif !isValidEmail(%s){\n", varName))
		sb.WriteString(fmt.Sprintf("\t\t\terrs=append(errs,%q)\n", vf.Name+" must be a valid email"))
		sb.WriteString("\t\t}\n")
	}
	if minArg := decArg(vf.Decorators, "min"); minArg != "" {
		sb.WriteString(fmt.Sprintf("\t\tif len(%s)<%s{\n", varName, minArg))
		sb.WriteString(fmt.Sprintf("\t\t\terrs=append(errs,%q)\n", vf.Name+" must be at least "+minArg+" characters"))
		sb.WriteString("\t\t}\n")
	}
	if maxArg := decArg(vf.Decorators, "max"); maxArg != "" {
		sb.WriteString(fmt.Sprintf("\t\tif len(%s)>%s{\n", varName, maxArg))
		sb.WriteString(fmt.Sprintf("\t\t\terrs=append(errs,%q)\n", vf.Name+" must be at most "+maxArg+" characters"))
		sb.WriteString("\t\t}\n")
	}
	if args := decArgs(vf.Decorators, "oneof"); len(args) > 0 {
		sb.WriteString(fmt.Sprintf("\t\tvalid_%s:=map[string]bool{", vf.Name))
		for i, a := range args {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("%q:true", a))
		}
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("\t\tif !valid_%s[%s]{\n", vf.Name, varName))
		sb.WriteString(fmt.Sprintf("\t\t\terrs=append(errs,%q)\n", vf.Name+" must be one of: "+strings.Join(args, ",")))
		sb.WriteString("\t\t}\n")
	}
}

func genNumericValidation(sb *strings.Builder, varName string, vf VouchField) {
	if minArg := decArg(vf.Decorators, "min"); minArg != "" {
		sb.WriteString(fmt.Sprintf("\t\tif %s<%s{\n", varName, minArg))
		sb.WriteString(fmt.Sprintf("\t\t\terrs=append(errs,%q)\n", vf.Name+" must be at least "+minArg))
		sb.WriteString("\t\t}\n")
	}
	if maxArg := decArg(vf.Decorators, "max"); maxArg != "" {
		sb.WriteString(fmt.Sprintf("\t\tif %s>%s{\n", varName, maxArg))
		sb.WriteString(fmt.Sprintf("\t\t\terrs=append(errs,%q)\n", vf.Name+" must be at most "+maxArg))
		sb.WriteString("\t\t}\n")
	}
}

func genParamParser(r RouteDecl) string {
	var sb strings.Builder
	pascal := toPascalCase(r.Handler)

	sb.WriteString(fmt.Sprintf("func parse%sParams(r *http.Request)%sParams{\n", pascal, pascal))
	sb.WriteString(fmt.Sprintf("\tvar params %sParams\n", pascal))

	if r.Grabit != nil {
		for _, w := range r.Grabit.Wheres {
			if w.Source.Kind == SourceQuery {
				pascalSrc := toPascalCase(w.Source.Value)
				sb.WriteString(fmt.Sprintf("\tif v:=r.URL.Query().Get(%q);v!=\"\"{\n", w.Source.Value))
				sb.WriteString(fmt.Sprintf("\t\tval:=v\n"))
				sb.WriteString(fmt.Sprintf("\t\tparams.%s=&val\n", pascalSrc))
				sb.WriteString("\t}\n")
			}
		}
		if r.Grabit.OrderBy != nil && r.Grabit.OrderBy.Kind == SourceQuery {
			v := r.Grabit.OrderBy.Value
			def := r.Grabit.OrderByDefault
			if def == "" {
				def = "created_at"
			}
			sb.WriteString(fmt.Sprintf("\tparams.%s=r.URL.Query().Get(%q)\n", toPascalCase(v), v))
			sb.WriteString(fmt.Sprintf("\tif params.%s==\"\"{\n", toPascalCase(v)))
			sb.WriteString(fmt.Sprintf("\t\tparams.%s=%q\n", toPascalCase(v), def))
			sb.WriteString("\t}\n")
		}
		if r.Grabit.OrderDir != nil && r.Grabit.OrderDir.Kind == SourceQuery {
			v := r.Grabit.OrderDir.Value
			def := r.Grabit.OrderDirDefault
			if def == "" {
				def = "desc"
			}
			sb.WriteString(fmt.Sprintf("\tparams.%s=r.URL.Query().Get(%q)\n", toPascalCase(v), v))
			sb.WriteString(fmt.Sprintf("\tif params.%s==\"\"{\n", toPascalCase(v)))
			sb.WriteString(fmt.Sprintf("\t\tparams.%s=%q\n", toPascalCase(v), def))
			sb.WriteString("\t}\n")
		}
		if r.Grabit.Limit != nil && r.Grabit.Limit.Kind == SourceQuery {
			v := r.Grabit.Limit.Value
			def := r.Grabit.LimitDefault
			if def == 0 {
				def = 20
			}
			sb.WriteString(fmt.Sprintf("\tif v:=r.URL.Query().Get(%q);v!=\"\"{\n", v))
			sb.WriteString("\t\tn,_:=strconv.Atoi(v)\n")
			sb.WriteString(fmt.Sprintf("\t\tparams.%s=n\n", toPascalCase(v)))
			sb.WriteString("\t}else{\n")
			sb.WriteString(fmt.Sprintf("\t\tparams.%s=%d\n", toPascalCase(v), def))
			sb.WriteString("\t}\n")
		}
		if r.Grabit.Offset != nil && r.Grabit.Offset.Kind == SourceQuery {
			v := r.Grabit.Offset.Value
			def := r.Grabit.OffsetDefault
			sb.WriteString(fmt.Sprintf("\tif v:=r.URL.Query().Get(%q);v!=\"\"{\n", v))
			sb.WriteString("\t\tn,_:=strconv.Atoi(v)\n")
			sb.WriteString(fmt.Sprintf("\t\tparams.%s=n\n", toPascalCase(v)))
			sb.WriteString("\t}else{\n")
			sb.WriteString(fmt.Sprintf("\t\tparams.%s=%d\n", toPascalCase(v), def))
			sb.WriteString("\t}\n")
		}
	}

	sb.WriteString("\treturn params\n")
	sb.WriteString("}\n\n")

	return sb.String()
}
