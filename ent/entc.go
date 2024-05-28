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
					for _, node := range g.Nodes {
						tag := edge.Annotation{StructTag: `json:"-"`}
						node.Annotations.Set(tag.Name(), tag)

						for _, field := range node.Fields {
							field.StructTag = fmt.Sprintf(`json:"%s,omitempty"`, camel(field.Name))
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
