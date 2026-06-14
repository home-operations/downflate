// Package talos pulls container images onto Talos Linux nodes via the machinery
// gRPC API, warming each node's image cache ahead of a PR merge.
package talos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/siderolabs/talos/pkg/machinery/api/common"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"

	"github.com/home-operations/downflate/internal/config"
)

// Puller pulls images onto a fixed set of Talos nodes.
type Puller struct {
	client      *client.Client
	nodes       []string
	driver      common.ContainerDriver
	namespace   common.ContainerdNamespace
	concurrency int
}

// New builds a Puller from the configured talosconfig. The client authenticates
// via the mTLS material embedded in the talosconfig context.
func New(ctx context.Context, cfg *config.Config) (*Puller, error) {
	opts := []client.OptionFunc{client.WithConfigFromFile(cfg.Talosconfig)}
	if cfg.TalosContext != "" {
		opts = append(opts, client.WithContextName(cfg.TalosContext))
	}
	c, err := client.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("talos client: %w", err)
	}

	nodes := cfg.Nodes
	if len(nodes) == 0 {
		if nodes, err = nodesFromConfig(cfg); err != nil {
			_ = c.Close()
			return nil, err
		}
	}
	if len(nodes) == 0 {
		_ = c.Close()
		return nil, errors.New("talos: no nodes configured (set DOWNFLATE_NODES or talosconfig nodes/endpoints)")
	}

	driver, ns := common.ContainerDriver_CRI, common.ContainerdNamespace_NS_CRI
	if cfg.Namespace == "system" {
		driver, ns = common.ContainerDriver_CONTAINERD, common.ContainerdNamespace_NS_SYSTEM
	}

	return &Puller{
		client:      c,
		nodes:       nodes,
		driver:      driver,
		namespace:   ns,
		concurrency: cfg.Concurrency,
	}, nil
}

// Close releases the underlying gRPC connection.
func (p *Puller) Close() error { return p.client.Close() }

// Nodes returns the node addresses this puller targets.
func (p *Puller) Nodes() []string { return p.nodes }

// Result records the outcome of pulling one image onto one node.
type Result struct {
	Node string
	Ref  string
	Err  error
}

// PullAll pulls every ref onto every node, bounded by the configured
// concurrency. Every (node, ref) pair is attempted regardless of individual
// failures; the returned slice has one entry per pair.
func (p *Puller) PullAll(ctx context.Context, refs []string) []Result {
	type job struct{ node, ref string }
	var jobs []job
	for _, n := range p.nodes {
		for _, r := range refs {
			jobs = append(jobs, job{n, r})
		}
	}

	results := make([]Result, len(jobs))
	sem := make(chan struct{}, p.concurrency)
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			err := p.pull(ctx, j.node, j.ref)
			results[i] = Result{Node: j.node, Ref: j.ref, Err: err}
		}()
	}
	wg.Wait()
	return results
}

// pull streams a single image pull to completion on one node. The machinery
// Pull RPC is server-streaming; the pull is complete once the stream closes.
func (p *Puller) pull(ctx context.Context, node, ref string) error {
	nctx := client.WithNodes(ctx, node)
	stream, err := p.client.ImageClient.Pull(nctx, &machine.ImageServicePullRequest{
		Containerd: &common.ContainerdInstance{
			Driver:    p.driver,
			Namespace: p.namespace,
		},
		ImageRef: ref,
	})
	if err != nil {
		return err
	}
	for {
		if _, err := stream.Recv(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// nodesFromConfig falls back to the talosconfig context's nodes, then endpoints.
func nodesFromConfig(cfg *config.Config) ([]string, error) {
	tc, err := clientconfig.Open(cfg.Talosconfig)
	if err != nil {
		return nil, fmt.Errorf("read talosconfig: %w", err)
	}
	name := cfg.TalosContext
	if name == "" {
		name = tc.Context
	}
	ctx := tc.Contexts[name]
	if ctx == nil {
		return nil, fmt.Errorf("talosconfig context %q not found", name)
	}
	if len(ctx.Nodes) > 0 {
		return ctx.Nodes, nil
	}
	return ctx.Endpoints, nil
}
