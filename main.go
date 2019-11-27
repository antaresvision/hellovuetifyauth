package main

import (
	"encoding/json"
	"fmt"
	"github.com/antaresvision/hellovuetifyauth/api"
	"github.com/antaresvision/hellovuetifyauth/db"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"net/http"
	"strings"
	"time"
)

type apiResponse struct {
	Message string		`json:"message"`
	TimeStamp time.Time `json:"time_stamp"`
}

func main() {
	sessions := map[string]string{}

	dataStore := db.NewConnection()
	defer dataStore.Close()

	srv := api.NewServer(dataStore)

	r := mux.NewRouter()

	r.Path("/greetings").Methods(http.MethodGet).HandlerFunc(GreetingsHandler)
	r.Path("/greetings/{name}").Methods(http.MethodGet).HandlerFunc(GreetingsHandler)

	r.Path("/login").Methods(http.MethodPost).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user := request.PostFormValue("user")
		password := request.PostFormValue("password")

		if user == "foo" && password == "bar" {
			encryptedUser := EncryptToBase64(user)
			http.SetCookie(writer, &http.Cookie{
				Name:       "myauth",
				Value:      encryptedUser,
			})
		} else {
			http.Error(writer, "invalid username/password", http.StatusBadRequest)
		}
	})

	r.Path("/login/token").Methods(http.MethodPost).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, password, ok := request.BasicAuth()
		if !ok {
			http.Error(writer, "invalid username/password", http.StatusBadRequest)
			return
		}

		if user == "foo" && password == "bar" {
			key := uuid.New().String()
			sessions[key]=user
			writer.Write([]byte(key))
		} else {
			http.Error(writer, "invalid username/password", http.StatusBadRequest)
		}
	})

	itemsRouter := r.PathPrefix("/items").Subrouter()
	itemsRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("prima dell'handler")

			var user = ""

			authCookie, err := r.Cookie("myauth")
			if err == nil {
				user = DecryptFromBase64(authCookie.Value)
			}

			if user == "" {
				reqToken := r.Header.Get("Authorization")
				splitToken := strings.Split(reqToken, "Bearer ")
				if len(splitToken) == 2 {
					reqToken = splitToken[1]
					user = sessions[reqToken]
				}
			}

			log.Println("User:", user)
			var auth = (user == "foo")

			if auth {
				next.ServeHTTP(w, r)

				encryptedUser := EncryptToBase64(user)
				http.SetCookie(w, &http.Cookie{
					Name:       "myauth",
					Value:      encryptedUser,
				})
			} else {
				log.Println("Not authorized")
				http.Error(w, "Auth required", http.StatusUnauthorized)
			}
			log.Println("dopo l'handler")
		})
	})

	itemsRouter.Path("/{id}").Methods(http.MethodGet).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("Get item handler")
		api.GetItemById(writer, request, dataStore)
	})
	itemsRouter.Path("/{id}").Methods(http.MethodDelete).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("Remove item handler")
		api.RemoveItemById(writer, request, dataStore)
	})
	itemsRouter.Path("/{id}").Methods(http.MethodPost).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("Save item handler")
		api.SaveItem(writer, request, dataStore)
	})
	itemsRouter.Methods(http.MethodGet).HandlerFunc(srv.GetAllItems)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8000", "http://localhost:8080"},
		AllowedHeaders: []string{"Authorization","Content-Type"},
		AllowCredentials: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	})
	http.ListenAndServe(":8000", c.Handler(r))
}

func GreetingsHandler(writer http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	name, found := variables["name"]
	if !found {
		name = "Generic User"
	}

	resp := fmt.Sprintf("Hello, Dear %s", name)

	fmt.Println(resp)

	jsonResp := apiResponse{
		Message:   resp,
		TimeStamp: time.Now(),
	}

	jsonBuffer, err := json.Marshal(jsonResp)
	if err != nil {
		fmt.Println(err)
		http.Error(writer, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(jsonBuffer)
	if err != nil {
		fmt.Println(err)
	}
}
