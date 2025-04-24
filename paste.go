package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

type Paste struct {
	Id            string    `schema:"-"`
	Content       string    `schema:"content,required"`
	SyntaxContent string    `schema:"-"`
	Syntax        string    `schema:"syntax,required"`
	Expire        int       `schema:"expire,required"`
	CreatedAt     time.Time `schema:"-"`
}

func getPaste(id string) (Paste, error) {
	var paste Paste
	err := db.QueryRow("SELECT id, content, syntax_content, syntax, expire, created_at FROM pastes WHERE id = $1", id).Scan(
		&paste.Id, &paste.Content, &paste.SyntaxContent, &paste.Syntax, &paste.Expire, &paste.CreatedAt,
	)

	// return err if it has expired
	if durationPaste(paste) <= (1*time.Second) && paste.Expire != 0 {
		return paste, errors.New("Paste does not exist")
	}

	return paste, err
}

func insertPaste(paste Paste) error {
	if lexers.Get(paste.Syntax) == nil {
		paste.Syntax = "disabled"
	}

	if paste.Syntax != "disabled" {
		buf := new(bytes.Buffer)

		formatter := html.New(html.Standalone(false), html.WithLineNumbers(true))
		style := styles.Get("github-dark")
		lexer := lexers.Get(paste.Syntax)
		paste.Syntax = lexer.Config().Name

		iterator, err := lexer.Tokenise(nil, paste.Content)
		if err != nil {
			return err
		}

		formatter.Format(buf, style, iterator)

		paste.SyntaxContent = buf.String()
	}

	_, err := db.Exec("INSERT INTO pastes (id, content, syntax_content, syntax, expire) VALUES ($1, $2, $3, $4, $5)",
		paste.Id, paste.Content, paste.SyntaxContent, paste.Syntax, paste.Expire,
	)

	return err
}

func createPasteTable(postgresql bool) {
	timestamp_type := "TIMESTAMP"
	if postgresql {
		timestamp_type = "TIMESTAMPTZ"
	}

	_, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS pastes (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			syntax_content TEXT,
			expire INT NOT NULL,
			syntax TEXT NOT NULL DEFAULT 'disabled',
			created_at %s NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`, timestamp_type))

	if err != nil {
		log.Fatal(err)
	}
}

func deleteOldPastes(postgresql bool) {
	if postgresql {
		db.Exec("DELETE FROM pastes WHERE CURRENT_TIMESTAMP > created_at + (expire * interval '1 second') AND expire != 0")
	} else {
		db.Exec("DELETE FROM pastes WHERE CURRENT_TIMESTAMP > DATETIME(created_at, expire || ' second') AND expire != 0")
	}

}

func randomId(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
