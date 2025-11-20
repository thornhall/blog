package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

func init() {
	styles.Register(chroma.MustNewStyle("custom-vscode", chroma.StyleEntries{
		chroma.Text:       "#ffffff",
		chroma.Background: "bg:#0e0e10",

		chroma.Comment: "#595958 italic",

		chroma.Punctuation: "#f5ce42",

		chroma.Keyword:          "#f77575 bold",
		chroma.KeywordNamespace: "#f77575 bold",
		chroma.Operator:         "#f77575 bold",

		chroma.NameFunction:         "#7ddafc",
		chroma.NameBuiltin:          "#44e7f9",
		chroma.NameVariable:         "#85f1fd italic",
		chroma.NameVariableInstance: "#85f1fd",
		chroma.NameAttribute:        "#61a1f0",
		chroma.NameProperty:         "#61a1f0",
		chroma.NameEntity:           "#44e7f9",

		chroma.NameClass:   "#6dfbdc",
		chroma.KeywordType: "#6dfbdc",
		chroma.String:      "#f5ce42",
		chroma.StringChar:  "#f5ce42",
		chroma.LiteralDate: "#f5ce42",

		chroma.Number:          "#f5ce42",
		chroma.KeywordConstant: "#f5ce42 bold",
		chroma.Literal:         "#f5ce42",
		chroma.StringInterpol:  "#96fea8",
		chroma.NameNamespace:   "#44e7f9",
		chroma.Error:           "#ff5555 bg:#110000",
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
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
			highlighting.NewHighlighting(
				highlighting.WithStyle("custom-vscode"),
				highlighting.WithFormatOptions(html.WithLineNumbers(true)),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(),
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
