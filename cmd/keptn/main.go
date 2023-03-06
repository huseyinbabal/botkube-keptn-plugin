package main

import (
	"botkube.io/plugins-example/cmd/keptn/internal"
	"github.com/hashicorp/go-plugin"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
)

func main() {
	executor.Serve(map[string]plugin.Plugin{
		"echo": &source.Plugin{
			Source: internal.NewSource("dev"),
		},
	})
}
