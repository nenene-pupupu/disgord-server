//go:build ignore

package main

import (
	"fmt"
	"log"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/edge"
)

func main() {
	err := entc.Generate("./schema", &gen.Config{
		Hooks: []gen.Hook{
			func(next gen.Generator) gen.Generator {
				return gen.GenerateFunc(func(g *gen.Graph) error {
					tag := edge.Annotation{StructTag: `json:"-"`}
					for _, n := range g.Nodes {
						n.Annotations.Set(tag.Name(), tag)

						for _, f := range n.Fields {
							f.StructTag = fmt.Sprintf(`json:"%s,omitempty"`, camel(f.Name))
						}
					}
					return next.Generate(g)
				})
			},
		},
	})
	if err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}
}

func camel(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if i > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, "")
}
