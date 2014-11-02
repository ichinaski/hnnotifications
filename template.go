package main

import (
	"errors"
	"fmt"
	"io"
	"text/template"
)

var (
	templates = make(map[string]*template.Template)
)

// init handles template initialization.
func init() {
	templates["info"] = template.Must(template.ParseFiles("templates/info.html"))
	templates["item_email"] = template.Must(template.ParseFiles("templates/item_email.html"))
	templates["activate_email"] = template.Must(template.ParseFiles("templates/activate_email.html"))
	templates["unsubscribe_email"] = template.Must(template.ParseFiles("templates/unsubscribe_email.html"))
}

// useTemplate applies the given data to the template, and writes the output to w
func useTemplate(name string, data interface{}, w io.Writer) error {
	t, ok := templates[name]
	if !ok {
		return errors.New(fmt.Sprintf("Template %s not found", name))
	}
	return t.Execute(w, data)
}
