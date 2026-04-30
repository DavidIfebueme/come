package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"come/internal/come"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "init":
		if len(os.Args) < 3 {
			fatal("usage: comec init <project-name>")
		}
		initProject(os.Args[2])
	case "add":
		if len(os.Args) < 3 || os.Args[2] != "feature" || len(os.Args) < 4 {
			fatal("usage: comec add feature <name>")
		}
		addFeature(os.Args[3])
	case "build":
		buildProject()
	case "run":
		buildProject()
		runProject()
	case "migrate":
		dir := "generated"
		if len(os.Args) >= 4 {
			dir = os.Args[3]
		}
		direction := "up"
		if len(os.Args) >= 3 {
			direction = os.Args[2]
		}
		runMigrate(dir, direction)
	case "seed":
		seedProject()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "come - the troll-opposite-of-go api language")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  init <name>           create a new come project")
	fmt.Fprintln(os.Stderr, "  add feature <name>    add a new feature slice")
	fmt.Fprintln(os.Stderr, "  build                 compile .come files to go")
	fmt.Fprintln(os.Stderr, "  run                   build and run the generated api")
	fmt.Fprintln(os.Stderr, "  migrate [up|down]     run database migrations")
	fmt.Fprintln(os.Stderr, "  seed                  run seed data")
}

func initProject(name string) {
	dir := name
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal(err.Error())
	}
	appContent := fmt.Sprintf(`nogo "%s"

pileup postgres "postgres://localhost/%s"

aura {
	port 8080
	read_timeout 10s
	write_timeout 10s
	idle_timeout 60s
}

unblockthehomies "*"
`, name, name)
	if err := os.WriteFile(filepath.Join(dir, "app.come"), []byte(appContent), 0o644); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("created project %s\n", name)
	fmt.Println("edit app.come to configure your api")
	fmt.Println("run 'comec add feature <name>' to add a feature")
}

func addFeature(name string) {
	dir := name
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal(err.Error())
		}
	}
	pascal := strings.ToUpper(name[:1]) + name[1:]
	content := fmt.Sprintf(`manifest %s {
	id         uuid      @primary @default(gen_random_uuid)
	created_at timestamp @default(now)
	updated_at timestamp @default(now) @auto_update

	spotlight id
}

yeet GET "/api/%s" list_%s {
	ward cors
	ward log

	grabit {
		from %s
		order_by query.sort_by @default(created_at)
		order_dir query.order @default(desc)
		limit query.limit @default(20)
		offset query.offset @default(0)
	}

	hurl 200 result
}

yeet POST "/api/%s" create_%s {
	ward cors
	ward log

	vouch {
	}

	grabit {
		insert %s body
	}

	hurl 201 result
	hurl 400 validation
}

yeet GET "/api/%s/:id" get_%s {
	ward cors
	ward log

	grabit {
		from %s
		where id == param.id
		one
	}

	hurl 200 result
	hurl 404 {status: "error", message: "not found"}
}

yeet PUT "/api/%s/:id" update_%s {
	ward cors
	ward log

	vouch {
	}

	grabit {
		update %s
		where id == param.id
		set body
	}

	hurl 200 result
	hurl 404 {status: "error", message: "not found"}
}

yeet DELETE "/api/%s/:id" delete_%s {
	ward cors
	ward log

	grabit {
		delete from %s
		where id == param.id
	}

	hurl 204
	hurl 404 {status: "error", message: "not found"}
}
`, pascal, name, name, pascal, name, name, pascal, name, name, pascal, name, name, pascal, name, name, pascal)

	filename := filepath.Join(dir, name+".come")
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		fatal(err.Error())
	}

	appFile := "app.come"
	data, err := os.ReadFile(appFile)
	if err != nil {
		fmt.Printf("created feature %s\n", name)
		fmt.Printf("add 'borrow \"./%s\"' to your app.come\n", name)
		return
	}
	if !strings.Contains(string(data), "borrow \"./"+name+"\"") {
		borrowLine := fmt.Sprintf("\nborrow \"./%s\"\n", name)
		if err := os.WriteFile(appFile, append(data, []byte(borrowLine)...), 0o644); err != nil {
			fatal(err.Error())
		}
	}
	fmt.Printf("created feature %s\n", name)
}

func buildProject() {
	rootDir := "."
	outDir := "generated"
	for i, arg := range os.Args {
		if arg == "-in" && i+1 < len(os.Args) {
			rootDir = os.Args[i+1]
		}
		if arg == "-out" && i+1 < len(os.Args) {
			outDir = os.Args[i+1]
		}
	}

	proj, err := come.ResolveProject(rootDir)
	if err != nil {
		fatal("resolve: " + err.Error())
	}
	if err := come.Check(proj); err != nil {
		fatal("check: " + err.Error())
	}
	files := come.Generate(proj)
	for path, content := range files {
		target := filepath.Join(outDir, path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			fatal(err.Error())
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			fatal(err.Error())
		}
	}

	for _, feat := range proj.Features {
		for _, seed := range feat.Seeds {
			src := seed.Path
			if !filepath.IsAbs(src) {
				src = filepath.Join(rootDir, src)
			}
			data, err := os.ReadFile(src)
			if err != nil {
				fatal("seed: " + err.Error())
			}
			dst := filepath.Join(outDir, feat.Name, filepath.Base(src))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				fatal(err.Error())
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				fatal(err.Error())
			}
		}
	}

	fmt.Printf("compiled come -> %s\n", outDir)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: go mod tidy failed: %v\n", err)
	}
}

func runProject() {
	outDir := "generated"
	runCmd := exec.Command("go", "run", "./cmd/server")
	runCmd.Dir = outDir
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Stdin = os.Stdin
	if err := runCmd.Run(); err != nil {
		fatal(err.Error())
	}
}

func runMigrate(dir, direction string) {
	migrationsDir := filepath.Join(dir, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		fatal("cannot read migrations: " + err.Error())
	}
	suffix := ".up.sql"
	if direction == "down" {
		suffix = ".down.sql"
	}
	fmt.Printf("running %s migrations from %s\n", direction, migrationsDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), suffix) {
			fmt.Printf("  %s\n", e.Name())
		}
	}
	fmt.Println("note: integrate with your preferred migration tool (golang-migrate, goose, etc.)")
}

func seedProject() {
	fmt.Println("seeding data...")
	fmt.Println("note: implement seed logic in your generated code")
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
