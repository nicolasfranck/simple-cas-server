package main

import (
	"log"
	"net/url"
	"os"
	"fmt"
	"flag"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/handlers"
	"html/template"
	"github.com/google/uuid"
)

type credentials struct{
	Error string
	Username string
	Password string
	Service  string
}

type ticketRecord struct {
	ticket string
	username string
	service string
}

var (
	loginTemplate = template.Must(template.ParseFiles("views/login.gohtml"))
	sessionStore *sessions.CookieStore = nil
	tickets = make([]*ticketRecord, 0)
	router *mux.Router = nil
)

func initSessionStore() {

	sessionSecret := []byte(uuid.NewString())
    sessionStore = sessions.NewCookieStore(sessionSecret)
    sessionStore.MaxAge(3600)
	baseUrl, _ := router.Get("home").URL()
    if baseUrl.Path != "" {
        sessionStore.Options.Path = baseUrl.Path
    } else {
		sessionStore.Options.Path = "/"
	}
    sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = baseUrl.Scheme == "https"
}

func getSession(req *http.Request) *sessions.Session{
	session, e := sessionStore.Get(req, "session")
	if e != nil {
		fmt.Fprintf(os.Stderr, "error while decoding session: %s", e.Error())
	}
	return session
}

func getSessionValue(session * sessions.Session, key string) string {
    valWithoutType := session.Values["user"]
	if valWithoutType == nil {
		return ""
	}
	val, ok := valWithoutType.(string)
	if !ok {
		return ""
	}
	return val
}

func getTicketRecordByUser(username string, service string) *ticketRecord{

	for _, t := range(tickets) {

		if t.username == username && t.service == service{
			return t
		}

	}

	return nil
}

func getTicketRecord(ticket string, service string) *ticketRecord{

	for _, t := range(tickets) {

		if t.service == service && t.ticket == ticket {

			return t

		}

	}

	return nil
}

func addTicketRecord(t *ticketRecord) {
	tickets = append(tickets, t)
}

func loginGet(w http.ResponseWriter, req *http.Request){

	service := req.FormValue("service")
	if service == "" {
		http.Error(w, "service not provided", 400)
		return
	}
	serviceUrl, errUrl := url.Parse(service)
	if errUrl != nil {
		http.Error(w, "invalid service url", 400)
		return
	}

	session := getSession(req)
	if session == nil {
		http.Error(w, "invalid session", 500)
		return
	}
	user    := getSessionValue(session, "user")

	if user != "" {
		tr := getTicketRecordByUser(user, service)
		if tr != nil {
			query := serviceUrl.Query()
			query.Set("ticket", tr.ticket)
			serviceUrl.RawQuery = query.Encode()
			http.Redirect(w, req, serviceUrl.String(), http.StatusFound)
			return
		}
	}

	loginTemplate.Execute(w, &credentials{ Service: service })
}

func loginPost(w http.ResponseWriter, req *http.Request){

	username := req.FormValue("username")
	password := req.FormValue("password")
	service  := req.FormValue("service")

	if service == "" {
        http.Error(w, "service not provided", 400)
        return
    }

	serviceUrl, errUrl := url.Parse(service)
    if errUrl != nil {
        http.Error(w, "invalid service url", 400)
        return
    }

	session := getSession(req)
	if session == nil {
        http.Error(w, "invalid session", 500)
        return
    }
	user    := getSessionValue(session, "user")

    if user != "" {
        tr := getTicketRecordByUser(user, service)
        if tr != nil {
			query := serviceUrl.Query()
            query.Set("ticket", tr.ticket)
            serviceUrl.RawQuery = query.Encode()
            http.Redirect(w, req, serviceUrl.String(), http.StatusFound)
            return
        }
    }

	if username == password {

		session.Values["user"] = username
		session.Save(req, w)
		ticket := uuid.NewString()
		addTicketRecord(&ticketRecord{
			ticket: ticket,
			username: username,
			service: service,
		})
		query := serviceUrl.Query()
		query.Set("ticket", ticket)
		serviceUrl.RawQuery = query.Encode()
		http.Redirect(w, req, serviceUrl.String(), http.StatusFound)
        return

	}

	loginTemplate.Execute(w, &credentials{ Username: username, Password: password, Service: service })
}

func logout(w http.ResponseWriter, req *http.Request){

	session := getSession(req)
	if session == nil {
        http.Error(w, "invalid session", 500)
        return
    }
	user := getSessionValue(session, "user")
	delete(session.Values, "user")
	session.Save(req, w)

	if user != "" {

		newTickets := make([]*ticketRecord, 0)
		for _, tr := range(tickets) {
			if tr.username != user {
				newTickets = append(newTickets, tr)
			}
		}
		tickets = newTickets

    }

	loginUrl,_ := router.Get("loginGet").URL()
	http.Redirect(w, req, loginUrl.String(), http.StatusFound)
}

func sendCasFailure(w http.ResponseWriter, code string, msg string) {

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(400)
	casError := `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
		<cas:authenticationFailure code="%s">%s</cas:authenticationFailure>
	</cas:serviceResponse>`
	fmt.Fprintf(w, casError, code, msg)

}

func sendCasSuccess(w http.ResponseWriter, user string) {

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(200)
	casError := `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
		<cas:authenticationSuccess>
			<cas:user>%s</cas:user>
		</cas:authenticationSuccess>
	</cas:serviceResponse>`
	fmt.Fprintf(w, casError, user)

}

func serviceValidate(w http.ResponseWriter, req *http.Request){

	ticket  := req.FormValue("ticket")
	service := req.FormValue("service")

    if service == "" {
		sendCasFailure(w, "NO_SERVICE", "no service provided")
        return
    }

	if ticket == "" {
		sendCasFailure(w, "NO_TICKET", "no ticket provided")
        return
	}

	tr := getTicketRecord(ticket, service)
	if tr == nil {
		sendCasFailure(w, "INVALID_TICKET", "invalid ticket")
		return
	}

	sendCasSuccess(w, tr.username)
}

func home(w http.ResponseWriter, req *http.Request) {

	w.WriteHeader(200)
	fmt.Fprintf(w, "ok")

}

func main(){

	router = mux.NewRouter()
	router.HandleFunc("/login", loginGet).Methods("GET").Name("loginGet")
	router.HandleFunc("/login", loginPost).Methods("POST").Name("loginPost")
	router.HandleFunc("/logout", logout).Methods("GET").Name("logout")
	router.HandleFunc("/serviceValidate", serviceValidate).Name("serviceValidate")
	router.HandleFunc("/", home).Name("home")
	fs := http.FileServer(http.Dir("./public/"))
    router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", fs)).Name("public")

	initSessionStore()

	bind := ":3000"
	flag.StringVar(&bind, "bind", ":3000", "bind")

	flag.Parse()

	log.Fatal(http.ListenAndServe(bind, handlers.LoggingHandler(os.Stdout, router)))

}
