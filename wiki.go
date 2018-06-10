package main

import (
	"html/template"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var upgrader = websocket.Upgrader{}

//Message is sent by Websocket
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

var templates = template.Must(template.ParseFiles("templates/chat.html"))
var validPath = regexp.MustCompile("^/(edit|save|view|chat)/([a-zA-Z0-9]+)$")
var viewLog ViewLog

// ChatLog is chat log
type ChatLog struct {
	ID      int
	Name    string
	Comment string
	Nice    int
}

func (c ChatLog) addNice() {
	c.Nice++
}

// ViewLog shows all Chatlog
type ViewLog struct {
	idcounter int
	Logs      []ChatLog
}

func (v ViewLog) Len() int {
	return len(v.Logs)
}

func (v ViewLog) Less(i, j int) bool {
	return v.Logs[i].Nice > v.Logs[j].Nice
}

func (v ViewLog) Swap(i, j int) {
	v.Logs[i], v.Logs[j] = v.Logs[j], v.Logs[i]
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

func (v ViewLog) addNice(id int) {
	v.Logs[id].Nice++
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
		viewLog.addLog(chatLog)
	} else if !strings.EqualFold(charLogIDStr, "") {
		id, _ := strconv.Atoi(charLogIDStr)
		log := viewLog.getLog(id)
		if log != nil {
			log.addNice()
			viewLog.addNice(id)
		}
	}
	sort.Sort(viewLog)
	renderChatTemplate(w, "chat", &viewLog)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/chat", http.StatusFound)
}

func renderChatTemplate(w http.ResponseWriter, tmpl string, d *ViewLog) {
	err := templates.ExecuteTemplate(w, tmpl+".html", d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()
	clients[ws] = true
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func main() {
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()

	//http.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("templates/"))))
	//http.HandleFunc("/", rootHandler)
	//http.HandleFunc("/chat/", chatHandler)
	log.Println("http server started on: 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
