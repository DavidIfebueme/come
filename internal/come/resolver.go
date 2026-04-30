package come

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveProject(rootDir string) (*Project, error) {
	appFile := filepath.Join(rootDir, "app.come")
	src, err := os.ReadFile(appFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read app.come: %w", err)
	}
	tokens, err := NewLexer(string(src)).Tokenize()
	if err != nil {
		return nil, fmt.Errorf("app.come: %w", err)
	}
	file, err := NewParser(tokens, string(src)).Parse()
	if err != nil {
		return nil, fmt.Errorf("app.come: %w", err)
	}
	proj := &Project{Aura: AuraDecl{Port: 8080}}
	for _, decl := range file.Declarations {
		switch d := decl.(type) {
		case AppDecl:
			proj.AppName = d.Name
		case DBDecl:
			proj.DBs = append(proj.DBs, d)
		case AuraDecl:
			proj.Aura = d
		case CORSDecl:
			proj.CORS = d
		case BouncerConfigDecl:
			proj.Bouncer = &d
		case VibesDecl:
			proj.Vibes = &d
		case BorrowDecl:
			feat, err := resolveFeature(rootDir, d.Path)
			if err != nil {
				return nil, fmt.Errorf("borrow %s: %w", d.Path, err)
			}
			proj.Features = append(proj.Features, *feat)
		case ManifestDecl:
			feat := Feature{Name: strings.ToLower(proj.AppName), Manifests: []ManifestDecl{d}}
			proj.Features = append(proj.Features, feat)
		case EnumDecl:
			if len(proj.Features) == 0 {
				proj.Features = append(proj.Features, Feature{Name: strings.ToLower(proj.AppName)})
			}
			proj.Features[len(proj.Features)-1].Enums = append(proj.Features[len(proj.Features)-1].Enums, d)
		case RouteDecl:
			if len(proj.Features) == 0 {
				proj.Features = append(proj.Features, Feature{Name: strings.ToLower(proj.AppName)})
			}
			proj.Features[len(proj.Features)-1].Routes = append(proj.Features[len(proj.Features)-1].Routes, d)
		case SpawnChaosDecl:
			if len(proj.Features) == 0 {
				proj.Features = append(proj.Features, Feature{Name: strings.ToLower(proj.AppName)})
			}
			proj.Features[len(proj.Features)-1].Seeds = append(proj.Features[len(proj.Features)-1].Seeds, d)
		case RawGoDecl:
			if len(proj.Features) == 0 {
				proj.Features = append(proj.Features, Feature{Name: strings.ToLower(proj.AppName)})
			}
			proj.Features[len(proj.Features)-1].RawGo = append(proj.Features[len(proj.Features)-1].RawGo, d)
		case ReshapeDecl:
			if len(proj.Features) == 0 {
				proj.Features = append(proj.Features, Feature{Name: strings.ToLower(proj.AppName)})
			}
			proj.Features[len(proj.Features)-1].Reshapes = append(proj.Features[len(proj.Features)-1].Reshapes, d)
		}
	}
	if len(proj.DBs) > 0 {
		proj.Driver = proj.DBs[0].Driver
	}
	for i := range proj.Features {
		if proj.Features[i].Name == "" {
			proj.Features[i].Name = fmt.Sprintf("feature_%d", i)
		}
	}
	return proj, nil
}

func resolveFeature(rootDir, relPath string) (*Feature, error) {
	cleanPath := strings.TrimPrefix(relPath, "./")
	featDir := filepath.Join(rootDir, cleanPath)
	entries, err := os.ReadDir(featDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory %s: %w", featDir, err)
	}
	var comeFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".come") {
			comeFiles = append(comeFiles, filepath.Join(featDir, e.Name()))
		}
	}
	if len(comeFiles) == 0 {
		return nil, fmt.Errorf("no .come files found in %s", featDir)
	}
	feat := &Feature{Name: filepath.Base(cleanPath)}
	for _, cf := range comeFiles {
		src, err := os.ReadFile(cf)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", cf, err)
		}
		tokens, err := NewLexer(string(src)).Tokenize()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", cf, err)
		}
		file, err := NewParser(tokens, string(src)).Parse()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", cf, err)
		}
		for _, decl := range file.Declarations {
			switch d := decl.(type) {
			case ManifestDecl:
				feat.Manifests = append(feat.Manifests, d)
			case EnumDecl:
				feat.Enums = append(feat.Enums, d)
			case RouteDecl:
				feat.Routes = append(feat.Routes, d)
			case SpawnChaosDecl:
				feat.Seeds = append(feat.Seeds, d)
			case RawGoDecl:
				feat.RawGo = append(feat.RawGo, d)
			case ReshapeDecl:
				feat.Reshapes = append(feat.Reshapes, d)
			}
		}
	}
	return feat, nil
}
