package come

import (
	"fmt"
	"strings"
)

func Check(proj *Project) error {
	if strings.TrimSpace(proj.AppName) == "" {
		return fmt.Errorf("missing nogo declaration")
	}
	if len(proj.DBs) == 0 {
		return fmt.Errorf("missing pileup declaration")
	}
	if strings.TrimSpace(proj.CORS.Origin) == "" {
		return fmt.Errorf("missing unblockthehomies declaration")
	}
	enumMap := map[string]*EnumDecl{}
	modelMap := map[string]*ManifestDecl{}
	for i := range proj.Features {
		f := &proj.Features[i]
		for j := range f.Enums {
			enumMap[f.Enums[j].Name] = &f.Enums[j]
		}
		for j := range f.Manifests {
			m := &f.Manifests[j]
			modelMap[m.Name] = m
			if err := checkManifest(m, enumMap); err != nil {
				return fmt.Errorf("manifest %s: %w", m.Name, err)
			}
		}
		for j := range f.Routes {
			if err := checkRoute(&f.Routes[j], modelMap, enumMap); err != nil {
				return fmt.Errorf("route %s %s %s: %w", f.Routes[j].Method, f.Routes[j].Path, f.Routes[j].Handler, err)
			}
		}
	}
	return nil
}

func checkManifest(m *ManifestDecl, enumMap map[string]*EnumDecl) error {
	hasPrimary := false
	for _, f := range m.Fields {
		for _, d := range f.Decorators {
			if d.Name == "primary" {
				if hasPrimary {
					return fmt.Errorf("multiple @primary fields in manifest %s", m.Name)
				}
				hasPrimary = true
			}
		}
		if f.Type.Kind == FieldEnum {
			if _, ok := enumMap[f.Type.Ref]; !ok {
				return fmt.Errorf("unknown enum %q referenced by field %s", f.Type.Ref, f.Name)
			}
		}
		if f.Type.Kind == FieldArray && f.Type.Items != nil && f.Type.Items.Kind == FieldEnum {
			if _, ok := enumMap[f.Type.Items.Ref]; !ok {
				return fmt.Errorf("unknown enum %q referenced by field %s", f.Type.Items.Ref, f.Name)
			}
		}
	}
	return nil
}

func checkRoute(r *RouteDecl, modelMap map[string]*ManifestDecl, enumMap map[string]*EnumDecl) error {
	if r.Grabit != nil {
		if _, ok := modelMap[r.Grabit.Model]; !ok {
			return fmt.Errorf("unknown model %q in grabit", r.Grabit.Model)
		}
	}
	if r.Vouch != nil {
		for _, vf := range r.Vouch.Fields {
			for _, d := range vf.Decorators {
				if d.Name == "oneof" && len(d.Args) == 0 {
					return fmt.Errorf("vouch field %s: @oneof requires at least one argument", vf.Name)
				}
			}
		}
	}
	if len(r.Hurls) == 0 {
		return fmt.Errorf("route has no hurl (response) declarations")
	}
	_ = enumMap
	return nil
}
