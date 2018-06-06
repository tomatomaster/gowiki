package main

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const pageStoredPath = "data/"
const separator = string(0x1e)

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

var viewLog ViewLog

// ChatLog is chat log
type ChatLog struct {
	ID      int
	Name    string
	Comment string
	Nice    int
}

func (c *ChatLog) addNice() {
	c.Nice++
}

// ViewLog shows all Chatlog
type ViewLog struct {
	idcounter int
	Logs      []ChatLog
}

func (v *ViewLog) getLog(id int) *ChatLog {
	for _, log := range v.Logs {
		if log.ID == id {
			return &log
		}
	}
	return nil
}

func (v *ViewLog) addLog(c ChatLog) *ViewLog {
	c.ID = v.idcounter
	v.Logs = append(v.Logs, c)
	v.idcounter++
	return v
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	comment := r.FormValue("chat")
	name := r.FormValue("name")
	charLogIDStr := r.FormValue("count")
	if !strings.EqualFold(comment, "") {
		if strings.EqualFold(name, "") {
			name = "名無しさん"
		}
		chatLog := ChatLog{Name: name, Comment: comment}
		fmt.Printf("add chatLog %v", chatLog)
		viewLog.addLog(chatLog)
	} else if !strings.EqualFold(charLogIDStr, "") {
		id, _ := strconv.Atoi(charLogIDStr)
		log := viewLog.getLog(id)
		if log != nil {
			log.addNice()
			fmt.Printf("%v", viewLog.Logs)
		}
	}
	renderChatTemplate(w, "chat", &viewLog)
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("Invalid Page Title")
	}
	return m[2], nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/chat", http.StatusFound)
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

func main() {
	http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates/"))))
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/chat/", chatHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
