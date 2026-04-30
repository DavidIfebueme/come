package come

import (
	"fmt"
	"strings"
)

func GenBabble(proj *Project, feat Feature, babble BabbleDecl) string {
	model := findModel(proj, babble.Model)
	if model == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("package " + feat.Name + "\n\nimport(\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"strconv\"\n")
	sb.WriteString("\t\"strings\"\n")
	sb.WriteString(fmt.Sprintf("\t\"%s/pkg/server\"\n", proj.AppName))
	if proj.Bouncer != nil {
		sb.WriteString(fmt.Sprintf("\t\"%s/pkg/auth\"\n", proj.AppName))
	}
	sb.WriteString(")\n\n")

	if proj.Bouncer != nil {
		sb.WriteString("var _=auth.JWTAuth{}\n\n")
	}

	sb.WriteString(genBabbleParser(babble, model))
	sb.WriteString(genBabbleHandler(proj, feat, babble, model))
	sb.WriteString(genBabbleSearchRepo(babble, model, proj))
	sb.WriteString(genBabbleSearchParams(babble))

	return sb.String()
}

func genBabbleParser(babble BabbleDecl, model *ManifestDecl) string {
	var sb strings.Builder

	sb.WriteString(`func parseNLQuery(q string)nlParams{
	tokens:=strings.Fields(strings.ToLower(q))
	var params nlParams
	seen:=map[string]int{}
	for i:=0;i<len(tokens);i++{
		tok:=tokens[i]
`)
	for _, rule := range babble.Rules {
		switch rule.Kind {
		case BabbleKeyword:
			sb.WriteString("\t\tif ")
			for j, w := range rule.Words {
				if j > 0 {
					sb.WriteString("||")
				}
				sb.WriteString(fmt.Sprintf("tok==%q", w))
			}
			sb.WriteString("{\n")
			for field, val := range rule.Filters {
				sb.WriteString(fmt.Sprintf("\t\t\tif prev,ok:=seen[%q];ok&&prev!=0&&seen[%q]!=0{\n", field, field))
				sb.WriteString(fmt.Sprintf("\t\t\t\tparams.%s=nil\n", toPascalCase(field)))
				sb.WriteString("\t\t\t} else {\n")
				sb.WriteString(fmt.Sprintf("\t\t\t\tparams.%s=ptrStr(%q)\n", toPascalCase(field), val))
				sb.WriteString(fmt.Sprintf("\t\t\t\tseen[%q]++\n", field))
				sb.WriteString("\t\t\t}\n")
			}
			sb.WriteString("\t\t}\n")
		case BabblePrefix:
			sb.WriteString("\t\tif ")
			for j, w := range rule.Words {
				if j > 0 {
					sb.WriteString("||")
				}
				sb.WriteString(fmt.Sprintf("tok==%q", w))
			}
			sb.WriteString("{\n")
			sb.WriteString("\t\t\tif i+1<len(tokens){\n")
			for field, valType := range rule.Filters {
				switch valType {
				case "int":
					sb.WriteString(fmt.Sprintf("\t\t\t\tif v,err:=strconv.Atoi(tokens[i+1]);err==nil{\n"))
					sb.WriteString(fmt.Sprintf("\t\t\t\t\tparams.%s=ptrStr(strconv.Itoa(v))\n", toPascalCase(field)))
					sb.WriteString("\t\t\t\t}\n")
				case "country":
					sb.WriteString(fmt.Sprintf("\t\t\t\tif code,ok:=countryLookup(tokens[i+1]);ok{\n"))
					sb.WriteString(fmt.Sprintf("\t\t\t\t\tparams.%s=ptrStr(code)\n", toPascalCase(field)))
					sb.WriteString("\t\t\t\t}\n")
				default:
					sb.WriteString(fmt.Sprintf("\t\t\t\tparams.%s=ptrStr(tokens[i+1])\n", toPascalCase(field)))
				}
			}
			sb.WriteString("\t\t\t\ti++\n")
			sb.WriteString("\t\t\t}\n")
			sb.WriteString("\t\t}\n")
		}
	}

	if len(babble.Ignore) > 0 {
		sb.WriteString("\t\tif ")
		for j, w := range babble.Ignore {
			if j > 0 {
				sb.WriteString("||")
			}
			sb.WriteString(fmt.Sprintf("tok==%q", w))
		}
		sb.WriteString("{\n")
		sb.WriteString("\t\t\tcontinue\n")
		sb.WriteString("\t\t}\n")
	}

	sb.WriteString("\t}\n")
	sb.WriteString("\treturn params\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func ptrStr(s string)*string{return &s}\n\n")

	sb.WriteString(genCountryLookup())

	return sb.String()
}

func genCountryLookup() string {
	countries := map[string]string{
		"nigeria": "NG", "angola": "AO", "kenya": "KE", "ghana": "GH",
		"south africa": "ZA", "uganda": "UG", "tanzania": "TZ", "ethiopia": "ET",
		"cameroon": "CM", "senegal": "SN", "mali": "ML", "benin": "BJ",
		"togo": "TG", "niger": "NE", "burkina faso": "BF", "ivory coast": "CI",
		"cote d'ivoire": "CI", "guinea": "GN", "sierra leone": "SL", "liberia": "LR",
		"gambia": "GM", "cape verde": "CV", "mauritania": "MR", "chad": "TD",
		"congo": "CG", "democratic republic of congo": "CD", "gabon": "GA",
		"equatorial guinea": "GQ", "central african republic": "CF", "rwanda": "RW",
		"burundi": "BI", "somalia": "SO", "djibouti": "DJ", "eritrea": "ER",
		"sudan": "SD", "south sudan": "SS", "egypt": "EG", "libya": "LY",
		"tunisia": "TN", "algeria": "DZ", "morocco": "MA", "zambia": "ZM",
		"zimbabwe": "ZW", "mozambique": "MZ", "malawi": "MW", "madagascar": "MG",
		"botswana": "BW", "namibia": "NA", "eswatini": "SZ", "lesotho": "LS",
		"comoros": "KM", "seychelles": "SC", "mauritius": "MU",
		"united states": "US", "usa": "US", "america": "US", "united kingdom": "GB",
		"uk": "GB", "britain": "GB", "england": "GB", "canada": "CA",
		"australia": "AU", "france": "FR", "germany": "DE", "brazil": "BR",
		"india": "IN", "china": "CN", "japan": "JP", "mexico": "MX",
		"argentina": "AR", "italy": "IT", "spain": "ES", "portugal": "PT",
		"netherlands": "NL", "belgium": "BE", "switzerland": "CH", "sweden": "SE",
		"norway": "NO", "denmark": "DK", "finland": "FI", "poland": "PL",
		"russia": "RU", "turkey": "TR", "saudi arabia": "SA", "uae": "AE",
		"dubai": "AE", "israel": "IL", "singapore": "SG", "south korea": "KR",
		"thailand": "TH", "vietnam": "VN", "indonesia": "ID", "philippines": "PH",
		"malaysia": "MY", "new zealand": "NZ", "ireland": "IE", "colombia": "CO",
		"peru": "PE", "chile": "CL", "venezuela": "VE", "cuban": "CU",
		"panama": "PA", "costa rica": "CR", "ecuador": "EC", "bolivia": "BO",
		"paraguay": "PY", "uruguay": "UY", "honduras": "HN", "guatemala": "GT",
		"el salvador": "SV", "nicaragua": "NI", "dominican republic": "DO",
		"jamaica": "JM", "haiti": "HT", "trinidad": "TT", "bahamas": "BS",
		"barbados": "BB", "grenada": "GD", "guyana": "GY", "suriname": "SR",
		"belize": "BZ", "iceland": "IS", "czech republic": "CZ", "czechia": "CZ",
		"greece": "GR", "hungary": "HU", "romania": "RO", "bulgaria": "BG",
		"croatia": "HR", "serbia": "RS", "ukraine": "UA", "austria": "AT",
		"slovakia": "SK", "slovenia": "SI", "lithuania": "LT", "latvia": "LV",
		"estonia": "EE", "cyprus": "CY", "luxembourg": "LU", "malta": "MT",
		"monaco": "MC", "andorra": "AD", "san marino": "SM", "vatican": "VA",
		"liechtenstein": "LI", "moldova": "MD", "albania": "AL",
		"north macedonia": "MK", "montenegro": "ME", "bosnia": "BA",
		"kosovo": "XK", "georgia": "GE", "armenia": "AM", "azerbaijan": "AZ",
		"kazakhstan": "KZ", "uzbekistan": "UZ", "turkmenistan": "TM",
		"kyrgyzstan": "KG", "tajikistan": "TJ", "mongolia": "MN",
		"nepal": "NP", "bhutan": "BT", "bangladesh": "BD", "myanmar": "MM",
		"cambodia": "KH", "laos": "LA", "brunei": "BN", "timor": "TL",
		"maldives": "MV", "sri lanka": "LK", "pakistan": "PK", "afghanistan": "AF",
		"iran": "IR", "iraq": "IQ", "syria": "SY", "jordan": "JO",
		"lebanon": "LB", "yemen": "YE", "oman": "OM", "qatar": "QA",
		"kuwait": "KW", "bahrain": "BH", "palestine": "PS",
	}

	var sb strings.Builder
	sb.WriteString("func countryLookup(name string)(string,bool){\n")
	sb.WriteString("\tm:=map[string]string{\n")
	keys := make([]string, 0, len(countries))
	for k := range countries {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if len(keys[j]) > len(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("\t\t%q:%q,\n", k, countries[k]))
	}
	sb.WriteString("\t}\n")
	sb.WriteString("\tnormalized:=strings.ToLower(strings.TrimSpace(name))\n")
	sb.WriteString("\tfor k,v:=range m{\n")
	sb.WriteString("\t\tif strings.Contains(normalized,k){\n")
	sb.WriteString("\t\t\treturn v,true\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn \"\",false\n")
	sb.WriteString("}\n\n")
	return sb.String()
}

func genBabbleHandler(proj *Project, feat Feature, babble BabbleDecl, model *ManifestDecl) string {
	var sb strings.Builder
	handlerName := "Search" + model.Name

	sb.WriteString(fmt.Sprintf("func (h *Handler)%s(w http.ResponseWriter,r *http.Request){\n", handlerName))

	if proj.Bouncer != nil {
		sb.WriteString("\tif h.auth!=nil{\n")
		sb.WriteString("\t\t_,err:=h.auth.Validate(r)\n")
		sb.WriteString("\t\tif err!=nil{\n")
		sb.WriteString("\t\t\tserver.Unauthorized(w)\n")
		sb.WriteString("\t\t\treturn\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t}\n")
	}

	sb.WriteString("\tq:=r.URL.Query().Get(\"q\")\n")
	sb.WriteString("\tif q==\"\"{\n")
	sb.WriteString("\t\tserver.Error(w,http.StatusBadRequest,\"missing query parameter 'q'\")\n")
	sb.WriteString("\t\treturn\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\tnlParams:=parseNLQuery(q)\n")
	sb.WriteString("\tif !nlParams.hasAny(){\n")
	sb.WriteString("\t\tw.Header().Set(\"Content-Type\",\"application/json\")\n")
	sb.WriteString("\t\tw.WriteHeader(http.StatusBadRequest)\n")
	sb.WriteString("\t\tjson.NewEncoder(w).Encode(map[string]string{\"status\":\"error\",\"message\":\"Unable to interpret query\"})\n")
	sb.WriteString("\t\treturn\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString(fmt.Sprintf("\tparams:=nlParams.to%sParams(r)\n\n", model.Name))

	sb.WriteString(fmt.Sprintf("\tresults,total,err:=h.repo.%s(r.Context(),params)\n", handlerName))
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\tserver.Error(w,http.StatusInternalServerError,\"internal server error\")\n")
	sb.WriteString("\t\treturn\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\tw.Header().Set(\"Content-Type\",\"application/json\")\n")
	sb.WriteString("\ttotalPages:=total/params.Limit\n")
	sb.WriteString("\tif total%params.Limit!=0{totalPages++}\n")
	sb.WriteString("\tvar prevPage *int\n")
	sb.WriteString("\tif params.Page>1{p:=params.Page-1;prevPage=&p}\n")
	sb.WriteString("\tvar nextPage *int\n")
	sb.WriteString("\tif params.Page<totalPages{n:=params.Page+1;nextPage=&n}\n")
	searchPath := "/api/" + toSnakeCase(babble.Model) + "/search"
	sb.WriteString(fmt.Sprintf("\tselfLink:=fmt.Sprintf(\"%s?page=%%d&limit=%%d\",params.Page,params.Limit)\n", searchPath))
	sb.WriteString("\tnextLink:=\"\"\n")
	sb.WriteString(fmt.Sprintf("\tif nextPage!=nil{nextLink=fmt.Sprintf(\"%s?page=%%d&limit=%%d\",*nextPage,params.Limit)}\n", searchPath))
	sb.WriteString("\tprevLink:=\"\"\n")
	sb.WriteString(fmt.Sprintf("\tif prevPage!=nil{prevLink=fmt.Sprintf(\"%s?page=%%d&limit=%%d\",*prevPage,params.Limit)}\n", searchPath))
	sb.WriteString("\tlinks:=map[string]any{\"self\":selfLink}\n")
	sb.WriteString("\tif nextLink!=\"\"{links[\"next\"]=nextLink}else{links[\"next\"]=nil}\n")
	sb.WriteString("\tif prevLink!=\"\"{links[\"prev\"]=prevLink}else{links[\"prev\"]=nil}\n")
	sb.WriteString("\tjson.NewEncoder(w).Encode(map[string]any{\"status\":\"success\",\"page\":params.Page,\"limit\":params.Limit,\"total\":total,\"total_pages\":totalPages,\"links\":links,\"data\":results})\n")
	sb.WriteString("}\n\n")

	return sb.String()
}

func genBabbleSearchParams(babble BabbleDecl) string {
	model := babble.Model
	var sb strings.Builder

	seen := map[string]bool{}
	for _, rule := range babble.Rules {
		for field := range rule.Filters {
			seen[field] = true
		}
	}

	sb.WriteString(fmt.Sprintf("type nlParams struct{\n"))
	for field := range seen {
		sb.WriteString(fmt.Sprintf("\t%s *string\n", toPascalCase(field)))
	}
	sb.WriteString("}\n\n")

	sb.WriteString("func (p nlParams)hasAny()bool{\n")
	sb.WriteString("\treturn ")
	first := true
	for field := range seen {
		if !first {
			sb.WriteString("||")
		}
		first = false
		sb.WriteString(fmt.Sprintf("p.%s!=nil", toPascalCase(field)))
	}
	if first {
		sb.WriteString("false")
	}
	sb.WriteString("\n}\n\n")

	sb.WriteString(fmt.Sprintf("func (p nlParams)to%sParams(r *http.Request)%sSearchParams{\n", model, model))
	sb.WriteString(fmt.Sprintf("\tvar params %sSearchParams\n", model))
	sb.WriteString("\tparams.Page=1\n")
	sb.WriteString("\tparams.Limit=10\n")
	sb.WriteString("\tif v:=r.URL.Query().Get(\"page\");v!=\"\"{\n")
	sb.WriteString("\t\tif n,err:=strconv.Atoi(v);err==nil&&n>0{\n")
	sb.WriteString("\t\t\tparams.Page=n\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif v:=r.URL.Query().Get(\"limit\");v!=\"\"{\n")
	sb.WriteString("\t\tif n,err:=strconv.Atoi(v);err==nil&&n>0&&n<=50{\n")
	sb.WriteString("\t\t\tparams.Limit=n\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	for field := range seen {
		sb.WriteString(fmt.Sprintf("\tif p.%s!=nil{\n", toPascalCase(field)))
		sb.WriteString(fmt.Sprintf("\t\tparams.%s=*p.%s\n", toPascalCase(field), toPascalCase(field)))
		sb.WriteString("\t}\n")
	}
	sb.WriteString("\treturn params\n")
	sb.WriteString("}\n\n")

	return sb.String()
}

func genBabbleSearchRepo(babble BabbleDecl, model *ManifestDecl, proj *Project) string {
	var sb strings.Builder
	handlerName := "Search" + model.Name
	tbl := tableName(babble.Model)

	sb.WriteString(fmt.Sprintf("func (r *Repository)%s(ctx context.Context,params %sSearchParams)([]%s,int,error){\n", handlerName, model.Name, model.Name))
	sb.WriteString("\tvar conditions []string\n")
	sb.WriteString("\tvar args []any\n")
	sb.WriteString("\targPos:=1\n\n")

	generated := map[string]bool{}
	for _, rule := range babble.Rules {
		for field, val := range rule.Filters {
			if generated[field] {
				continue
			}
			generated[field] = true

			switch field {
			case "min_age":
				sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(field)))
				sb.WriteString("\t\tconditions=append(conditions,fmt.Sprintf(\"age >= $%d\",argPos))\n")
				sb.WriteString(fmt.Sprintf("\t\tif v,err:=strconv.Atoi(params.%s);err==nil{\n", toPascalCase(field)))
				sb.WriteString("\t\t\targs=append(args,v)\n")
				sb.WriteString("\t\t} else {\n")
				sb.WriteString(fmt.Sprintf("\t\t\targs=append(args,params.%s)\n", toPascalCase(field)))
				sb.WriteString("\t\t}\n")
				sb.WriteString("\t\targPos++\n")
				sb.WriteString("\t}\n")
			case "max_age":
				sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(field)))
				sb.WriteString("\t\tconditions=append(conditions,fmt.Sprintf(\"age <= $%d\",argPos))\n")
				sb.WriteString(fmt.Sprintf("\t\tif v,err:=strconv.Atoi(params.%s);err==nil{\n", toPascalCase(field)))
				sb.WriteString("\t\t\targs=append(args,v)\n")
				sb.WriteString("\t\t} else {\n")
				sb.WriteString(fmt.Sprintf("\t\t\targs=append(args,params.%s)\n", toPascalCase(field)))
				sb.WriteString("\t\t}\n")
				sb.WriteString("\t\targPos++\n")
				sb.WriteString("\t}\n")
			case "min_gender_probability":
				sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(field)))
				sb.WriteString("\t\tconditions=append(conditions,fmt.Sprintf(\"gender_probability >= $%d\",argPos))\n")
				sb.WriteString(fmt.Sprintf("\t\tif v,err:=strconv.ParseFloat(params.%s,64);err==nil{\n", toPascalCase(field)))
				sb.WriteString("\t\t\targs=append(args,v)\n")
				sb.WriteString("\t\t} else {\n")
				sb.WriteString(fmt.Sprintf("\t\t\targs=append(args,params.%s)\n", toPascalCase(field)))
				sb.WriteString("\t\t}\n")
				sb.WriteString("\t\targPos++\n")
				sb.WriteString("\t}\n")
			case "min_country_probability":
				sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(field)))
				sb.WriteString("\t\tconditions=append(conditions,fmt.Sprintf(\"country_probability >= $%d\",argPos))\n")
				sb.WriteString(fmt.Sprintf("\t\tif v,err:=strconv.ParseFloat(params.%s,64);err==nil{\n", toPascalCase(field)))
				sb.WriteString("\t\t\targs=append(args,v)\n")
				sb.WriteString("\t\t} else {\n")
				sb.WriteString(fmt.Sprintf("\t\t\targs=append(args,params.%s)\n", toPascalCase(field)))
				sb.WriteString("\t\t}\n")
				sb.WriteString("\t\targPos++\n")
				sb.WriteString("\t}\n")
			default:
				if rule.Kind == BabblePrefix && val == "country" {
					colName := toSnakeCase(field)
					sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(field)))
					sb.WriteString(fmt.Sprintf("\t\tconditions=append(conditions,fmt.Sprintf(\"%s = $%%d\",argPos))\n", colName))
					sb.WriteString(fmt.Sprintf("\t\targs=append(args,params.%s)\n", toPascalCase(field)))
					sb.WriteString("\t\targPos++\n")
					sb.WriteString("\t}\n")
				} else {
					colName := toSnakeCase(field)
					sb.WriteString(fmt.Sprintf("\tif params.%s!=\"\"{\n", toPascalCase(field)))
					sb.WriteString(fmt.Sprintf("\t\tconditions=append(conditions,fmt.Sprintf(\"%s = $%%d\",argPos))\n", colName))
					sb.WriteString(fmt.Sprintf("\t\targs=append(args,params.%s)\n", toPascalCase(field)))
					sb.WriteString("\t\targPos++\n")
					sb.WriteString("\t}\n")
				}
			}
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

	sb.WriteString("\toffset:=(params.Page-1)*params.Limit\n\n")

	sb.WriteString(fmt.Sprintf("\tquery:=r.db.Rebind(fmt.Sprintf(\"SELECT %s FROM %s \"+whereClause+\" ORDER BY created_at DESC LIMIT $%%d OFFSET $%%d\",argPos,argPos+1))\n", modelColumns(model), tbl))
	sb.WriteString("\targs=append(args,params.Limit,offset)\n\n")

	sb.WriteString("\trows,err:=r.db.QueryContext(ctx,query,args...)\n")
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\treturn nil,0,err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tdefer rows.Close()\n\n")

	sb.WriteString(fmt.Sprintf("\tresults,err:=scan%sRows(rows)\n", model.Name))
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\treturn nil,0,err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn results,total,nil\n")
	sb.WriteString("}\n\n")

	return sb.String()
}
