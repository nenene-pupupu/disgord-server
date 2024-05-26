//go:build ignore

package main

import (
	"fmt"
	"log"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/edge"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
							field.StructTag = fmt.Sprintf(`json:"%s,omitempty"`, snakeToCamel(field.Name))
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

func snakeToCamel(s string) string {
	caser := cases.Title(language.Und)

	parts := strings.Split(s, "_")
	for i, part := range parts {
		if i > 0 {
			parts[i] = caser.String(part)
		}
	}

	return strings.Join(parts, "")
}
