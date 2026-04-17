package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type routeDef struct {
	Method  string
	Path    string
	Handler string
}

type program struct {
	Name        string
	StoragePath string
	CORSOrigin  string
	Genderize   string
	Agify       string
	Nationalize string
	Routes      []routeDef
}

func main() {
	in := flag.String("in", "", "path to .come source file")
	out := flag.String("out", "", "output directory for generated go service")
	flag.Parse()

	if strings.TrimSpace(*in) == "" || strings.TrimSpace(*out) == "" {
		fatal("usage: comec -in <file.come> -out <output-dir>")
	}

	src, err := os.ReadFile(*in)
	if err != nil {
		fatal(err.Error())
	}

	prog, err := parseProgram(string(src))
	if err != nil {
		fatal(err.Error())
	}

	if err := validateProgram(prog); err != nil {
		fatal(err.Error())
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err.Error())
	}

	goModPath := filepath.Join(*out, "go.mod")
	mainPath := filepath.Join(*out, "main.go")

	goModContent := renderGoMod(prog)
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		fatal(err.Error())
	}

	mainContent := renderMain(prog)
	if err := os.WriteFile(mainPath, []byte(mainContent), 0o644); err != nil {
		fatal(err.Error())
	}

	fmt.Printf("compiled %s -> %s\n", *in, *out)
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func parseProgram(src string) (program, error) {
	prog := program{}
	scanner := bufio.NewScanner(strings.NewReader(src))
	lineNumber := 0

	vibecodeRe := regexp.MustCompile(`^vibecode\s+"([^"]+)"\s*$`)
	stashRe := regexp.MustCompile(`^stash\s+"([^"]+)"\s*$`)
	dripcorsRe := regexp.MustCompile(`^dripcors\s+"([^"]+)"\s*$`)
	teaRe := regexp.MustCompile(`^tea\s+(genderize|agify|nationalize)\s+"([^"]+)"\s*$`)
	cookRe := regexp.MustCompile(`^cook\s+(post|get|delete)\s+"([^"]+)"\s+([A-Za-z_][A-Za-z0-9_]*)\s*$`)

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		if m := vibecodeRe.FindStringSubmatch(line); m != nil {
			prog.Name = m[1]
			continue
		}
		if m := stashRe.FindStringSubmatch(line); m != nil {
			prog.StoragePath = m[1]
			continue
		}
		if m := dripcorsRe.FindStringSubmatch(line); m != nil {
			prog.CORSOrigin = m[1]
			continue
		}
		if m := teaRe.FindStringSubmatch(line); m != nil {
			service := m[1]
			url := m[2]
			switch service {
			case "genderize":
				prog.Genderize = url
			case "agify":
				prog.Agify = url
			case "nationalize":
				prog.Nationalize = url
			}
			continue
		}
		if m := cookRe.FindStringSubmatch(line); m != nil {
			prog.Routes = append(prog.Routes, routeDef{
				Method:  strings.ToUpper(m[1]),
				Path:    m[2],
				Handler: m[3],
			})
			continue
		}

		return program{}, fmt.Errorf("syntax error on line %d: %s", lineNumber, line)
	}

	if err := scanner.Err(); err != nil {
		return program{}, err
	}

	return prog, nil
}

func validateProgram(prog program) error {
	if strings.TrimSpace(prog.Name) == "" {
		return errors.New("missing vibecode declaration")
	}
	if strings.TrimSpace(prog.CORSOrigin) == "" {
		return errors.New("missing dripcors declaration")
	}
	if len(prog.Routes) == 0 {
		return errors.New("missing cook route declarations")
	}

	if isStage1Program(prog) {
		if strings.TrimSpace(prog.StoragePath) == "" {
			return errors.New("missing stash declaration for stage1 profile mode")
		}
		if strings.TrimSpace(prog.Genderize) == "" || strings.TrimSpace(prog.Agify) == "" || strings.TrimSpace(prog.Nationalize) == "" {
			return errors.New("missing one or more tea upstream declarations for stage1 profile mode")
		}
	}

	return nil
}

func isStage1Program(prog program) bool {
	required := map[string]bool{
		"POST /api/profiles create_profile":       false,
		"GET /api/profiles/:id get_profile":       false,
		"GET /api/profiles list_profiles":         false,
		"DELETE /api/profiles/:id delete_profile": false,
	}

	for _, r := range prog.Routes {
		key := fmt.Sprintf("%s %s %s", r.Method, r.Path, r.Handler)
		if _, ok := required[key]; ok {
			required[key] = true
		}
	}

	for _, ok := range required {
		if !ok {
			return false
		}
	}

	return true
}

func renderGoMod(prog program) string {
	if isStage1Program(prog) {
		return generatedStage1GoMod
	}
	return generatedGenericGoMod
}

func renderMain(prog program) string {
	if isStage1Program(prog) {
		replacer := strings.NewReplacer(
			"{{APP_NAME}}", escapeForDoubleQuotedGoString(prog.Name),
			"{{STASH_PATH}}", escapeForDoubleQuotedGoString(prog.StoragePath),
			"{{CORS_ORIGIN}}", escapeForDoubleQuotedGoString(prog.CORSOrigin),
			"{{GENDERIZE_URL}}", escapeForDoubleQuotedGoString(prog.Genderize),
			"{{AGIFY_URL}}", escapeForDoubleQuotedGoString(prog.Agify),
			"{{NATIONALIZE_URL}}", escapeForDoubleQuotedGoString(prog.Nationalize),
		)
		return replacer.Replace(generatedStage1MainTemplate)
	}

	routeEntries := renderGenericRouteEntries(prog.Routes)
	handlerSwitch := renderGenericHandlerSwitch(prog.Routes)
	handlerFuncs := renderGenericHandlerFunctions(prog.Routes)
	replacer := strings.NewReplacer(
		"{{APP_NAME}}", escapeForDoubleQuotedGoString(prog.Name),
		"{{CORS_ORIGIN}}", escapeForDoubleQuotedGoString(prog.CORSOrigin),
		"{{ROUTE_ENTRIES}}", routeEntries,
		"{{HANDLER_SWITCH}}", handlerSwitch,
		"{{HANDLER_FUNCTIONS}}", handlerFuncs,
	)
	return replacer.Replace(generatedGenericMainTemplate)
}

func escapeForDoubleQuotedGoString(s string) string {
	s = strings.ReplaceAll(s, `\\`, `\\\\`)
	s = strings.ReplaceAll(s, `"`, `\\"`)
	return s
}

func renderGenericRouteEntries(routes []routeDef) string {
	lines := make([]string, 0, len(routes))
	for _, route := range routes {
		line := fmt.Sprintf("\t\t{Method: \"%s\", Pattern: \"%s\", Handler: \"%s\"},",
			escapeForDoubleQuotedGoString(route.Method),
			escapeForDoubleQuotedGoString(route.Path),
			escapeForDoubleQuotedGoString(route.Handler),
		)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderGenericHandlerFunctions(routes []routeDef) string {
	seen := map[string]bool{}
	ordered := []string{}
	for _, route := range routes {
		if seen[route.Handler] {
			continue
		}
		seen[route.Handler] = true
		ordered = append(ordered, route.Handler)
	}

	parts := make([]string, 0, len(ordered))
	for _, handlerName := range ordered {
		parts = append(parts, fmt.Sprintf(`func (a *app) %s(w http.ResponseWriter, r *http.Request, params map[string]string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var parsedBody any
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &parsedBody); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	query := map[string]string{}
	for key, values := range r.URL.Query() {
		if len(values) == 0 {
			continue
		}
		query[key] = values[len(values)-1]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"handler": "%s",
			"method":  r.Method,
			"path":    r.URL.Path,
			"params":  params,
			"query":   query,
			"body":    parsedBody,
		},
	})
}`,
			handlerName,
			escapeForDoubleQuotedGoString(handlerName),
		))
	}

	return strings.Join(parts, "\n\n")
}

func renderGenericHandlerSwitch(routes []routeDef) string {
	seen := map[string]bool{}
	ordered := []string{}
	for _, route := range routes {
		if seen[route.Handler] {
			continue
		}
		seen[route.Handler] = true
		ordered = append(ordered, route.Handler)
	}

	lines := make([]string, 0, len(ordered))
	for _, handlerName := range ordered {
		lines = append(lines, fmt.Sprintf("\tcase \"%s\":\n\t\ta.%s(w, r, params)\n\t\treturn nil",
			escapeForDoubleQuotedGoString(handlerName),
			handlerName,
		))
	}
	return strings.Join(lines, "\n")
}

const generatedGenericGoMod = `module generated/comeapi

go 1.23.0
`

const generatedStage1GoMod = `module generated/comeapi

go 1.23.0

require (
	github.com/google/uuid v1.6.0
	modernc.org/sqlite v1.34.5
)
`

const generatedGenericMainTemplate = `package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type route struct {
	Method  string
	Pattern string
	Handler string
}

type app struct {
	name       string
	corsOrigin string
	routes     []route
}

func main() {
	a := &app{
		name:       "{{APP_NAME}}",
		corsOrigin: "{{CORS_ORIGIN}}",
		routes: []route{
{{ROUTE_ENTRIES}}
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", a.withCORS(a.serve))
	addr := ":8080"
	log.Printf("%s listening on %s", a.name, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (a *app) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", a.corsOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (a *app) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "success",
			"data": map[string]any{
				"name": a.name,
			},
		})
		return
	}

	for _, route := range a.routes {
		if r.Method != route.Method {
			continue
		}
		params, ok := matchPath(route.Pattern, r.URL.Path)
		if !ok {
			continue
		}
		if err := a.dispatch(route.Handler, w, r, params); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeError(w, http.StatusNotFound, "route not found")
}

func (a *app) dispatch(handler string, w http.ResponseWriter, r *http.Request, params map[string]string) error {
	switch handler {
{{HANDLER_SWITCH}}
	default:
		return fmt.Errorf("handler %s not found", handler)
	}
}

func matchPath(pattern, actual string) (map[string]string, bool) {
	pattern = strings.TrimSpace(pattern)
	actual = strings.TrimSpace(actual)
	pattern = strings.Trim(pattern, "/")
	actual = strings.Trim(actual, "/")

	if pattern == "" && actual == "" {
		return map[string]string{}, true
	}

	patternSegments := strings.Split(pattern, "/")
	actualSegments := strings.Split(actual, "/")
	if len(patternSegments) != len(actualSegments) {
		return nil, false
	}

	params := map[string]string{}
	for index := range patternSegments {
		ps := patternSegments[index]
		as := actualSegments[index]
		if strings.HasPrefix(ps, ":") {
			key := strings.TrimPrefix(ps, ":")
			if key == "" {
				return nil, false
			}
			params[key] = as
			continue
		}
		if ps != as {
			return nil, false
		}
	}

	return params, true
}

{{HANDLER_FUNCTIONS}}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]any{
		"status":  "error",
		"message": message,
	})
}
`

const generatedStage1MainTemplate = `package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type app struct {
	db         *sql.DB
	httpClient *http.Client
	genderize  string
	agify      string
	nationalize string
	corsOrigin string
}

type profile struct {
	ID                 string    ` + "`json:\"id\"`" + `
	Name               string    ` + "`json:\"name\"`" + `
	Gender             string    ` + "`json:\"gender\"`" + `
	GenderProbability  float64   ` + "`json:\"gender_probability\"`" + `
	SampleSize         int       ` + "`json:\"sample_size\"`" + `
	Age                int       ` + "`json:\"age\"`" + `
	AgeGroup           string    ` + "`json:\"age_group\"`" + `
	CountryID          string    ` + "`json:\"country_id\"`" + `
	CountryProbability float64   ` + "`json:\"country_probability\"`" + `
	CreatedAt          time.Time ` + "`json:\"created_at\"`" + `
	NameLower          string    ` + "`json:\"-\"`" + `
}

type listItem struct {
	ID        string ` + "`json:\"id\"`" + `
	Name      string ` + "`json:\"name\"`" + `
	Gender    string ` + "`json:\"gender\"`" + `
	Age       int    ` + "`json:\"age\"`" + `
	AgeGroup  string ` + "`json:\"age_group\"`" + `
	CountryID string ` + "`json:\"country_id\"`" + `
}

type genderizeResponse struct {
	Gender      *string ` + "`json:\"gender\"`" + `
	Probability float64 ` + "`json:\"probability\"`" + `
	Count       int     ` + "`json:\"count\"`" + `
}

type agifyResponse struct {
	Age *int ` + "`json:\"age\"`" + `
}

type nationalizeResponse struct {
	Country []struct {
		CountryID   string  ` + "`json:\"country_id\"`" + `
		Probability float64 ` + "`json:\"probability\"`" + `
	} ` + "`json:\"country\"`" + `
}

func main() {
	name := "{{APP_NAME}}"
	storage := "{{STASH_PATH}}"
	cors := "{{CORS_ORIGIN}}"

	db, err := setupDB(storage)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	a := &app{
		db: db,
		httpClient: &http.Client{Timeout: 8 * time.Second},
		genderize: "{{GENDERIZE_URL}}",
		agify: "{{AGIFY_URL}}",
		nationalize: "{{NATIONALIZE_URL}}",
		corsOrigin: cors,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/profiles", a.withCORS(a.handleProfilesCollection))
	mux.HandleFunc("/api/profiles/", a.withCORS(a.handleProfilesResource))
	mux.HandleFunc("/healthz", a.withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]string{"name": name}})
	}))

	addr := ":8080"
	log.Printf("%s listening on %s", name, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func setupDB(path string) (*sql.DB, error) {
	abs := path
	if !filepath.IsAbs(path) {
		p, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		abs = p
	}

	db, err := sql.Open("sqlite", abs)
	if err != nil {
		return nil, err
	}

	schema := ` + "`" + `
	CREATE TABLE IF NOT EXISTS profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		name_lower TEXT NOT NULL UNIQUE,
		gender TEXT NOT NULL,
		gender_probability REAL NOT NULL,
		sample_size INTEGER NOT NULL,
		age INTEGER NOT NULL,
		age_group TEXT NOT NULL,
		country_id TEXT NOT NULL,
		country_probability REAL NOT NULL,
		created_at TEXT NOT NULL
	);
	` + "`" + `

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func (a *app) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", a.corsOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (a *app) handleProfilesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.createProfile(w, r)
	case http.MethodGet:
		a.listProfiles(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (a *app) handleProfilesResource(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "Profile not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.getProfile(w, r, id)
	case http.MethodDelete:
		a.deleteProfile(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (a *app) createProfile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rawName, ok := obj["name"]
	if !ok {
		writeError(w, http.StatusBadRequest, "Missing or empty name")
		return
	}

	name, ok := rawName.(string)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "Invalid type")
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Missing or empty name")
		return
	}

	nameLower := strings.ToLower(name)
	existing, err := a.getProfileByNameLower(r.Context(), nameLower)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Profile already exists",
			"data":    profileToJSON(existing),
		})
		return
	}

	prof, err := a.buildProfileFromUpstream(r.Context(), name, nameLower)
	if err != nil {
		var upstreamErr *upstreamInvalidError
		if errors.As(err, &upstreamErr) {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("%s returned an invalid response", upstreamErr.APIName))
			return
		}
		writeError(w, http.StatusBadGateway, "Upstream or server failure")
		return
	}

	if err := a.insertProfile(r.Context(), prof); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			existing, getErr := a.getProfileByNameLower(r.Context(), nameLower)
			if getErr == nil {
				writeJSON(w, http.StatusOK, map[string]any{
					"status":  "success",
					"message": "Profile already exists",
					"data":    profileToJSON(existing),
				})
				return
			}
		}
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "success",
		"data":   profileToJSON(prof),
	})
}

func (a *app) getProfile(w http.ResponseWriter, r *http.Request, id string) {
	prof, err := a.getProfileByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Profile not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   profileToJSON(prof),
	})
}

func (a *app) listProfiles(w http.ResponseWriter, r *http.Request) {
	filters := map[string]string{
		"gender":     strings.ToLower(strings.TrimSpace(r.URL.Query().Get("gender"))),
		"country_id": strings.ToLower(strings.TrimSpace(r.URL.Query().Get("country_id"))),
		"age_group":  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("age_group"))),
	}

	base := "SELECT id, name, gender, age, age_group, country_id FROM profiles"
	where := []string{}
	args := []any{}
	if filters["gender"] != "" {
		where = append(where, "LOWER(gender) = ?")
		args = append(args, filters["gender"])
	}
	if filters["country_id"] != "" {
		where = append(where, "LOWER(country_id) = ?")
		args = append(args, filters["country_id"])
	}
	if filters["age_group"] != "" {
		where = append(where, "LOWER(age_group) = ?")
		args = append(args, filters["age_group"])
	}

	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at ASC"

	rows, err := a.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	items := []listItem{}
	for rows.Next() {
		var item listItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Gender, &item.Age, &item.AgeGroup, &item.CountryID); err != nil {
			writeError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"count":  len(items),
		"data":   items,
	})
}

func (a *app) deleteProfile(w http.ResponseWriter, r *http.Request, id string) {
	res, err := a.db.ExecContext(r.Context(), "DELETE FROM profiles WHERE id = ?", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	affected, err := res.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "Profile not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type upstreamInvalidError struct {
	APIName string
}

func (e *upstreamInvalidError) Error() string {
	return e.APIName + " invalid response"
}

func (a *app) buildProfileFromUpstream(ctx context.Context, name, nameLower string) (profile, error) {
	genderizeURL := withNameQuery(a.genderize, name)
	agifyURL := withNameQuery(a.agify, name)
	nationalizeURL := withNameQuery(a.nationalize, name)

	var gRes genderizeResponse
	if err := a.getJSON(ctx, genderizeURL, &gRes); err != nil {
		return profile{}, err
	}
	if gRes.Gender == nil || strings.TrimSpace(*gRes.Gender) == "" || gRes.Count == 0 {
		return profile{}, &upstreamInvalidError{APIName: "Genderize"}
	}

	var aRes agifyResponse
	if err := a.getJSON(ctx, agifyURL, &aRes); err != nil {
		return profile{}, err
	}
	if aRes.Age == nil {
		return profile{}, &upstreamInvalidError{APIName: "Agify"}
	}

	var nRes nationalizeResponse
	if err := a.getJSON(ctx, nationalizeURL, &nRes); err != nil {
		return profile{}, err
	}
	if len(nRes.Country) == 0 {
		return profile{}, &upstreamInvalidError{APIName: "Nationalize"}
	}

	sort.Slice(nRes.Country, func(i, j int) bool {
		return nRes.Country[i].Probability > nRes.Country[j].Probability
	})
	top := nRes.Country[0]

	id, err := uuid.NewV7()
	if err != nil {
		return profile{}, err
	}

	createdAt := time.Now().UTC().Truncate(time.Second)
	return profile{
		ID:                 id.String(),
		Name:               name,
		NameLower:          nameLower,
		Gender:             strings.ToLower(strings.TrimSpace(*gRes.Gender)),
		GenderProbability:  gRes.Probability,
		SampleSize:         gRes.Count,
		Age:                *aRes.Age,
		AgeGroup:           classifyAge(*aRes.Age),
		CountryID:          strings.ToUpper(strings.TrimSpace(top.CountryID)),
		CountryProbability: top.Probability,
		CreatedAt:          createdAt,
	}, nil
}

func classifyAge(age int) string {
	if age <= 12 {
		return "child"
	}
	if age <= 19 {
		return "teenager"
	}
	if age <= 59 {
		return "adult"
	}
	return "senior"
}

func withNameQuery(baseURL, name string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + "?name=" + url.QueryEscape(name)
	}
	q := u.Query()
	q.Set("name", name)
	u.RawQuery = q.Encode()
	return u.String()
}

func (a *app) getJSON(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

func (a *app) insertProfile(ctx context.Context, p profile) error {
	_, err := a.db.ExecContext(ctx, ` + "`" + `
		INSERT INTO profiles (
			id, name, name_lower, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	` + "`" + `,
		p.ID,
		p.Name,
		p.NameLower,
		p.Gender,
		p.GenderProbability,
		p.SampleSize,
		p.Age,
		p.AgeGroup,
		p.CountryID,
		p.CountryProbability,
		p.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (a *app) getProfileByID(ctx context.Context, id string) (profile, error) {
	row := a.db.QueryRowContext(ctx, ` + "`" + `
		SELECT id, name, name_lower, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at
		FROM profiles WHERE id = ?
	` + "`" + `, id)
	return scanProfile(row)
}

func (a *app) getProfileByNameLower(ctx context.Context, nameLower string) (profile, error) {
	row := a.db.QueryRowContext(ctx, ` + "`" + `
		SELECT id, name, name_lower, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at
		FROM profiles WHERE name_lower = ?
	` + "`" + `, nameLower)
	return scanProfile(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProfile(row rowScanner) (profile, error) {
	var p profile
	var createdAtRaw string
	if err := row.Scan(
		&p.ID,
		&p.Name,
		&p.NameLower,
		&p.Gender,
		&p.GenderProbability,
		&p.SampleSize,
		&p.Age,
		&p.AgeGroup,
		&p.CountryID,
		&p.CountryProbability,
		&createdAtRaw,
	); err != nil {
		return profile{}, err
	}
	t, err := time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return profile{}, err
	}
	p.CreatedAt = t.UTC()
	return p, nil
}

func profileToJSON(p profile) map[string]any {
	return map[string]any{
		"id":                  p.ID,
		"name":                p.Name,
		"gender":              p.Gender,
		"gender_probability":  p.GenderProbability,
		"sample_size":         p.SampleSize,
		"age":                 p.Age,
		"age_group":           p.AgeGroup,
		"country_id":          p.CountryID,
		"country_probability": p.CountryProbability,
		"created_at":          p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]any{
		"status":  "error",
		"message": message,
	})
}
`
