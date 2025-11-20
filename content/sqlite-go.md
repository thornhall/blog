---
title: Implementing SQLite in Go + A Super Duper Long Long title + a long word like endometriosis
slug: sqlite-go
date: Nov 19, 2025
category: Engineering
excerpt: Why sticking with a single file database might actually be the smartest architectural decision you make this year.
---

Most developers immediately reach for PostgreSQL when starting a new project. It's the default choice. But for 99% of projects, **SQLite is actually faster, cheaper, and easier to maintain.**

## The Myth of Concurrency

People think SQLite can't handle concurrent writes. With `WAL Mode` (Write-Ahead Logging), this is largely solved for most read-heavy workloads.

> "SQLite is not a toy database. It is a production-grade engine used by Apple, Google, and Facebook."

## Code Example

Here is how you initialize it in Go:

```go
package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// --- CUSTOM THEME DEFINITION ---
func init() {
	// We register a new style named "custom-vscode" to match your JSON config.
	styles.Register(chroma.MustNewStyle("custom-vscode", chroma.StyleEntries{
		// Global Defaults
		chroma.Text:       "#ffffff",    // Default foreground
		chroma.Background: "bg:#0e0e10", // Keep your dark site background

		// Comments ("comment": "#595958 italic")
		chroma.Comment: "#595958 italic",

		// Punctuation ("punctuation": "#ffffff")
		chroma.Punctuation: "#ffffff",
		chroma.Operator:    "#ffffff",

		// Keywords ("keyword": "#f77575 bold")
		chroma.Keyword:          "#f77575 bold",
		chroma.KeywordNamespace: "#f77575 bold", // package, import

		// Functions ("support.function": "#7ddafc")
		chroma.NameFunction: "#7ddafc",
		chroma.NameBuiltin:  "#7ddafc", // print, len, etc.

		// Variables & Parameters
		// "parameter": "#85f1fd italic"
		// "support.variable": "#2889dd bold"
		chroma.NameVariable:         "#85f1fd italic",
		chroma.NameVariableInstance: "#85f1fd",
		chroma.NameAttribute:        "#61a1f0", // Attributes like HTML properties

		chroma.NameClass: "#6dfbdc",

		// Strings ("string": "#f5ce42")
		chroma.String:      "#f5ce42",
		chroma.StringChar:  "#f5ce42",
		chroma.LiteralDate: "#f5ce42",

		// Numbers & Constants ("number": "#f5ce42", "constant.language": "#f5ce42 bold")
		chroma.Number:          "#f5ce42",
		chroma.KeywordConstant: "#f5ce42 bold", // true, false, nil
		chroma.Literal:         "#f5ce42",

		// Special formatting
		// "string.format": "#96fea8"
		chroma.StringInterpol: "#96fea8",

		// "namespace": "#44e7f9"
		chroma.NameNamespace: "#44e7f9", // package names

		chroma.Error: "#ff5555 bg:#110000",
	}))
}

type Post struct {
	Title    string
	Slug     string
	Date     string
	Category string
	Excerpt  string
	Body     template.HTML
	Views    int
	Likes    int
}

type PageData struct {
	Title string
	Posts []Post
}

func main() {
	// 1. Configure Goldmark (Markdown Parser)
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
			highlighting.NewHighlighting(
				// USE YOUR NEW CUSTOM STYLE HERE
				highlighting.WithStyle("custom-vscode"),
				highlighting.WithFormatOptions(
				// Optional: Add line numbers if you want that "IDE" look
				// html.WithLineNumbers(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	// 2. Read all .md files from the content folder
	files, err := os.ReadDir("./content")
	if err != nil {
		log.Fatal("Could not read content directory:", err)
	}

	var posts []Post

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".md" {
			continue
		}

		// Read the file content
		source, err := os.ReadFile("./content/" + file.Name())
		if err != nil {
			log.Fatal(err)
		}

		// 3. Convert Markdown to HTML & Extract Meta
		var buf bytes.Buffer
		context := parser.NewContext()
		if err := md.Convert(source, &buf, parser.WithContext(context)); err != nil {
			log.Fatal(err)
		}

		// Extract Metadata (Frontmatter)
		metaData := meta.Get(context)

		// Helper to safely get string values from the map
		getString := func(key string) string {
			if v, ok := metaData[key]; ok {
				return fmt.Sprintf("%v", v)
			}
			return ""
		}

		p := Post{
			Title:    getString("title"),
			Slug:     getString("slug"),
			Date:     getString("date"),
			Category: getString("category"),
			Excerpt:  getString("excerpt"),
			Body:     template.HTML(buf.String()), // The converted HTML
		}

		posts = append(posts, p)
	}

	// 4. Prepare Templates
	data := PageData{
		Title: "The Obsidian Log",
		Posts: posts,
	}

	tmplIndex, err := template.ParseFiles("templates/layout.html", "templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	tmplPost, err := template.ParseFiles("templates/layout.html", "templates/post.html")
	if err != nil {
		log.Fatal(err)
	}

	// 5. Build Index Page
	f, err := os.Create("public/index.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := tmplIndex.Execute(f, data); err != nil {
		log.Fatal(err)
	}
	f.Close()
	log.Println("Generated: public/index.html")

	// 6. Build Article Pages
	for _, post := range posts {
		filename := "public/" + post.Slug + ".html"
		f, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}

		if err := tmplPost.Execute(f, post); err != nil {
			log.Fatal(err)
		}
		f.Close()
		log.Println("Generated:", filename)
	}
}

```