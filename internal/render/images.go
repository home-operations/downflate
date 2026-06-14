package render

import (
	"sort"

	"github.com/home-operations/flate/pkg/image"
	"github.com/home-operations/flate/pkg/orchestrator"
)

// ChangedImage is a container image the PR introduces to the head rendering.
type ChangedImage struct {
	Ref        string // full image reference present on the head side
	Name       string // repository portion
	Version    string // tag or digest on the head side
	OldVersion string // base-side version of the same repo, "" if newly added
}

// Changed returns the images present on the head side whose full reference is
// not present on the base side — i.e. newly added images and images whose
// tag/digest changed. These are the references worth pre-pulling before merge.
func Changed(base, head *orchestrator.Result) []ChangedImage {
	baseRefs := collect(base)
	headRefs := collect(head)

	// Map base repository name → version, to annotate changed images.
	baseByName := make(map[string]string, len(baseRefs))
	for ref := range baseRefs {
		name, ver := image.Split(ref)
		baseByName[name] = ver
	}

	var out []ChangedImage
	for ref := range headRefs {
		if _, unchanged := baseRefs[ref]; unchanged {
			continue
		}
		name, ver := image.Split(ref)
		out = append(out, ChangedImage{
			Ref:        ref,
			Name:       name,
			Version:    ver,
			OldVersion: baseByName[name],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out
}

// collect gathers the set of distinct image references across every rendered
// manifest in a result. A nil result yields an empty set.
func collect(res *orchestrator.Result) map[string]struct{} {
	refs := make(map[string]struct{})
	if res == nil {
		return refs
	}
	for _, docs := range res.Manifests {
		for _, doc := range docs {
			for _, ref := range image.Extract(doc) {
				refs[ref] = struct{}{}
			}
		}
	}
	return refs
}
