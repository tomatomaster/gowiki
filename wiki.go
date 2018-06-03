package main

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const pageStoredPath = "data/"

var templates = template.Must(template.ParseFiles("templates/edit.html", "templates/view.html", "templates/chat.html"))
var validPath = regexp.MustCompile("^/(edit|save|view|chat)/([a-zA-Z0-9]+)$")

// Page is wikipage
type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile(pageStoredPath+filename, p.Body, 0600)
}

// ChatLog is chat log
type ChatLog struct {
	Name    string
	Comment string
	Date    string
}

// ViewLog shows all Chatlog
type ViewLog struct {
	Logs []string
}

func (c ChatLog) saveLog() error {
	f, err := os.OpenFile(pageStoredPath+"chatLog", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	logFormat := fmt.Sprintf("%s【%s】%s\n", c.Date, c.Name, c.Comment)
	_, err = f.Write([]byte(logFormat))
	return err
}

func (c ChatLog) readAllLog() *ViewLog {
	logs, err := ioutil.ReadFile(pageStoredPath + "chatLog")
	if err != nil {
		log.Fatal(err)
	}
	viewLog := new(ViewLog)
	for _, line := range strings.Split(string(logs), "\n") {
		viewLog.Logs = append(viewLog.Logs, line)
	}
	viewLog.Logs = viewLog.Logs[:len(viewLog.Logs)-1]

	return viewLog
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	chatLog := ChatLog{Name: r.FormValue("name"), Comment: r.FormValue("chat"), Date: time.Now().Format(time.Stamp)}
	chatLog.saveLog()
	viewLog := chatLog.readAllLog()
	renderChatTemplate(w, "chat", viewLog)
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("Invalid Page Title")
	}
	return m[2], nil
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	title, err := getTitle(w, r)
	if err != nil {
		return
	}
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	title, err := getTitle(w, r)
	if err != nil {
		return
	}
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err = p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	title, err := getTitle(w, r)
	if err != nil {
		return
	}
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderChatTemplate(w http.ResponseWriter, tmpl string, d *ViewLog) {
	err := templates.ExecuteTemplate(w, tmpl+".html", d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, error := ioutil.ReadFile(pageStoredPath + filename)
	if error != nil {
		return nil, error
	}
	return &Page{Title: title, Body: body}, nil
}

func main() {
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates/"))))
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/chat/", chatHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/save/", saveHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
