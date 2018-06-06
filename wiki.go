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

var idCounter int
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
	Logs []ChatLog
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
	v.Logs = append(v.Logs, c)
	idCounter++
	return v
}

func (c ChatLog) saveLog() error {
	f, err := os.OpenFile(pageStoredPath+"chatLog", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	logFormat := fmt.Sprintf("%d%s%s%s%s%s%d\n", c.ID, separator, c.Name, separator, c.Comment, separator, c.Nice)
	_, err = f.Write([]byte(logFormat))
	return err
}

func readAllLog() *ViewLog {
	logs, err := ioutil.ReadFile(pageStoredPath + "chatlog")
	if err != nil {
		log.Fatal(err)
	}
	viewLog := new(ViewLog)
	if len(logs) == 0 {
		return viewLog
	}
	for _, line := range strings.Split(string(logs), "\n") {
		if strings.EqualFold(line, "") {
			break
		}
		data := strings.Split(line, separator)
		idStr := data[0]
		idInt, _ := strconv.Atoi(idStr)
		name := data[1]
		comment := data[2]
		niceStr := data[3]
		niceInt, _ := strconv.Atoi(niceStr)
		viewLog.Logs = append([]ChatLog{ChatLog{ID: idInt, Name: name, Comment: comment, Nice: niceInt}}, viewLog.Logs...)
	}
	viewLog.Logs = viewLog.Logs[:len(viewLog.Logs)]
	return viewLog
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	comment := r.FormValue("chat")
	name := r.FormValue("name")
	charLogIDStr := r.FormValue("count")
	if !strings.EqualFold(comment, "") {
		if strings.EqualFold(name, "") {
			name = "名無しさん"
		}
		chatLog := ChatLog{ID: idCounter, Name: name, Comment: comment}
		fmt.Printf("add chatLog %v", chatLog)
		viewLog.addLog(chatLog)
		idCounter++
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
