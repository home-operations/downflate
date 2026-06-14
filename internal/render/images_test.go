package render

import (
	"testing"

	"github.com/home-operations/flate/pkg/manifest"
	"github.com/home-operations/flate/pkg/orchestrator"
)

// deployment builds a minimal manifest doc with a container image, as flate's
// image.Extract would see after rendering.
func deployment(name, image string) map[string]any {
	return map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": name},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{"name": name, "image": image},
					},
				},
			},
		},
	}
}

func result(docs ...map[string]any) *orchestrator.Result {
	m := map[manifest.NamedResource][]map[string]any{}
	for _, d := range docs {
		name := d["metadata"].(map[string]any)["name"].(string)
		key := manifest.NamedResource{Kind: "Deployment", Name: name}
		m[key] = append(m[key], d)
	}
	return &orchestrator.Result{Manifests: m}
}

func refs(images []ChangedImage) map[string]ChangedImage {
	out := make(map[string]ChangedImage, len(images))
	for _, im := range images {
		out[im.Ref] = im
	}
	return out
}

func TestChanged(t *testing.T) {
	base := result(
		deployment("app", "ghcr.io/app:1.0.0"), // unchanged
		deployment("api", "ghcr.io/api:2.0.0"), // tag will change
		deployment("old", "ghcr.io/old:1.0.0"), // removed in head
	)
	head := result(
		deployment("app", "ghcr.io/app:1.0.0"), // unchanged → excluded
		deployment("api", "ghcr.io/api:2.1.0"), // changed → included
		deployment("new", "ghcr.io/new:0.1.0"), // added → included
	)

	got := refs(Changed(base, head))

	if len(got) != 2 {
		t.Fatalf("want 2 changed images, got %d: %v", len(got), got)
	}
	if _, ok := got["ghcr.io/app:1.0.0"]; ok {
		t.Errorf("unchanged image should be excluded")
	}
	if _, ok := got["ghcr.io/old:1.0.0"]; ok {
		t.Errorf("removed image should not be pulled")
	}
	api, ok := got["ghcr.io/api:2.1.0"]
	if !ok {
		t.Fatalf("changed image ghcr.io/api:2.1.0 missing")
	}
	if api.OldVersion != "2.0.0" || api.Version != "2.1.0" {
		t.Errorf("api version annotation: old=%q new=%q", api.OldVersion, api.Version)
	}
	added, ok := got["ghcr.io/new:0.1.0"]
	if !ok {
		t.Fatalf("added image ghcr.io/new:0.1.0 missing")
	}
	if added.OldVersion != "" {
		t.Errorf("added image should have empty OldVersion, got %q", added.OldVersion)
	}
}

func TestChangedNilSides(t *testing.T) {
	if got := Changed(nil, nil); len(got) != 0 {
		t.Fatalf("nil results should yield no images, got %v", got)
	}
}
