package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"strings"
	"strconv"
	"encoding/hex"
	"crypto/sha256"
	"database/sql"
	"eve/rsa"
	"eve/errors"
	"eve/torrent"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_"github.com/go-sql-driver/mysql"
)

var keys rsa.Keys
var quit bool

func JsonResponse(response interface{}, w http.ResponseWriter) {
	json, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func ValidateMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("authorization")
		if authorizationHeader != "" {
			bearerToken := strings.Split(authorizationHeader, " ")
			if len(bearerToken) == 2 {
				token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("There was an error")
					}
					return keys.PrivateKey(), nil
				})
				if err != nil {
					fmt.Println(err.Error())
					e := errors.Error{"Error while parsing token", http.StatusBadRequest}
					w.WriteHeader(e.HttpStatus)
					JsonResponse(e, w)
					return
				}
				if token.Valid {
					context.Set(r, "claims", token.Claims.(jwt.MapClaims))
					next(w, r)
				} else {
					e := errors.Error{"Invalid Token", http.StatusBadRequest}
					w.WriteHeader(e.HttpStatus)
					JsonResponse(e, w)
					return
				}
			}
		} else {
			e := errors.Error{"An authorization header is required", http.StatusBadRequest}
			w.WriteHeader(e.HttpStatus)
			JsonResponse(e, w)
			return
		}
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	type JwtToken struct {
		Token string `json:"token"`
	}
	type User struct {
		Username string `json: "username"`
		Password string `json: "password"`
	}
	var user User
	fmt.Println("Login")
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		e := errors.Error{"Error reading data", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	sha_256 := sha256.New()
	sha_256.Write([]byte(user.Password))
	user.Password = hex.EncodeToString(sha_256.Sum(nil))
	db, err := sql.Open("mysql", "eve:Tightwithay2016$@/eve")
	if err != nil {
		e := errors.Error{"Error accessing database", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	rows, err := db.Query("SELECT id, user, pass FROM users WHERE user = \"" + user.Username + "\"")
	if err != nil {
		e := errors.Error{"Error accessing database", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	if !rows.Next() {
		e := errors.Error{"Invalid Credentials", http.StatusBadRequest}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	var uid int
	var uuser string	
	var upass string
	rows.Scan(&uid, &uuser, &upass)
	if user.Password != upass {
		fmt.Println("Invalid Credentials")
		e := errors.Error{"Invalid Credentials", http.StatusBadRequest}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	claims := &jwt.StandardClaims {
		Issuer: "Eve",
		IssuedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		Id: strconv.Itoa(uid), //user id
		Subject: uuser, //username
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(keys.PrivateKey())
	if err != nil {
		e := errors.Error{"Error while singing token", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	fmt.Println("Successful")
	JsonResponse(JwtToken{tokenString}, w)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	type User struct {
		Username string `json: "username"`
		Password string `json: "password"`
		Email string `json: "email"`
		FirstName string `json: "firstname"`
		LastName string `json: "lastname"`
		Phone string `json: "phone"`
		Address string `json: "address"`
		Address2 string `json: "address2"`
		City string `json: "city"`
		State string `json: "state"`
		Zip string `json: "zip"`
	}
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		e := errors.Error{"Error reading data", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	db, err := sql.Open("mysql", "eve:Tightwithay2016$@/eve")
	if err != nil {
		e := errors.Error{"Error accessing database", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	rows, err := db.Query("SELECT id FROM users WHERE user = \"" + user.Username + "\"")
	if err != nil {
		e := errors.Error{"Error accessing database", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	if rows.Next() {
		fmt.Println(err)
		e := errors.Error{"Username is taken", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	stmt, err := db.Prepare("INSERT users SET user=?,pass=?,first_name=?,last_name=?,email=?,phone=?,address=?,address2=?,city=?,state=?,zip=?")
	if err != nil {
		e := errors.Error{"Error accessing database", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	sha_256 := sha256.New()
	sha_256.Write([]byte(user.Password))
	user.Password = hex.EncodeToString(sha_256.Sum(nil))
	res, err := stmt.Exec(user.Username, user.Password, user.FirstName, user.LastName, user.Email, user.Phone, user.Address, user.Address2, user.City, user.State, user.Zip)
	if err != nil {
		fmt.Println(err)
		e := errors.Error{"Error adding user", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	id, _ := res.LastInsertId()
	fmt.Println("User created with id " + strconv.Itoa(int(id)))
	db.Close()
}

func ClaimsHandler(w http.ResponseWriter, r *http.Request) {
	claims := context.Get(r, "claims").(jwt.MapClaims)
	JsonResponse(claims, w)
}

func TorrentHandler(w http.ResponseWriter, r *http.Request) {
	torrents, err := torrent.GetRunning()
	if err != nil {
		e := errors.Error{"Error getting torrent data", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	JsonResponse(torrents, w)
}

func TorrentSearchHandler(w http.ResponseWriter, r *http.Request) {
	var post torrent.Torrent
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		e := errors.Error{"Error reading data", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	results, ok, e := torrent.Search(post)
	if !ok {
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	JsonResponse(results, w)
	return
	/*
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(results)
	return
	*/
}

func TorrentAddHandler(w http.ResponseWriter, r *http.Request) {
	var  tor torrent.Torrent
	err := json.NewDecoder(r.Body).Decode(&tor)
	if err != nil {
		e := errors.Error{"Error reading data", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	claims := context.Get(r, "claims").(jwt.MapClaims)
	tor.User = claims["sub"].(string)
	tor.UserId, _ = strconv.Atoi(claims["jti"].(string))
	t, ok, e := torrent.Add(tor)
	if !ok {
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	JsonResponse(t, w)
}

func TorrentInfoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.FormValue("id")
	torrent, ok, err := torrent.Info(id)
	if !ok {
		w.WriteHeader(err.HttpStatus)
		JsonResponse(err, w)
		return
	}
	JsonResponse(torrent, w)
}

func TorrentPauseHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.FormValue("id")
	if id == "" {
		e := errors.Error{"Missing Id", http.StatusBadRequest}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
	}
	if ok, err := torrent.Pause(id); !ok {
		w.WriteHeader(err.HttpStatus)
		JsonResponse(err, w)
		return
	}
}

func TorrentResumeHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.FormValue("id")
	if id == "" {
		e := errors.Error{"Missing Id", http.StatusBadRequest}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
	}
	if ok, err := torrent.Resume(id); !ok {
		w.WriteHeader(err.HttpStatus)
		JsonResponse(err, w)
		return
	}
}

func TorrentRemoveHandler(w http.ResponseWriter, r *http.Request) {
	var  tor torrent.Torrent
	err := json.NewDecoder(r.Body).Decode(&tor)
	if err != nil {
		e := errors.Error{"Error reading data", http.StatusInternalServerError}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
		return
	}
	fmt.Println(tor.Id)
	if tor.Id == "" {
		e := errors.Error{"Missing Id", http.StatusBadRequest}
		w.WriteHeader(e.HttpStatus)
		JsonResponse(e, w)
	}
	if ok, err := torrent.Remove(tor.Id); !ok {
		w.WriteHeader(err.HttpStatus)
		JsonResponse(err, w)
		return
	}
}

func main() {
	fmt.Println("Generating RSA keys")
	keys.InitKeys()
	fmt.Println("Initializing...")
	quit = false
	go torrent.Start(&quit)
	fmt.Println("Booting server...")
	router := mux.NewRouter()
	router.HandleFunc("/authenticate", LoginHandler).Methods("POST")
	router.HandleFunc("/register", RegisterHandler).Methods("POST")
	router.HandleFunc("/claims", ValidateMiddleware(ClaimsHandler)).Methods("GET")
	router.HandleFunc("/torrent", ValidateMiddleware(TorrentHandler)).Methods("GET")
	router.HandleFunc("/torrent/search", ValidateMiddleware(TorrentSearchHandler)).Methods("POST")
	router.HandleFunc("/torrent/add", ValidateMiddleware(TorrentAddHandler)).Methods("POST")
	router.HandleFunc("/torrent/info", ValidateMiddleware(TorrentInfoHandler)).Methods("GET")
	router.HandleFunc("/torrent/pause", ValidateMiddleware(TorrentPauseHandler)).Methods("POST")
	router.HandleFunc("/torrent/resume", ValidateMiddleware(TorrentResumeHandler)).Methods("POST")
	router.HandleFunc("/torrent/remove", ValidateMiddleware(TorrentRemoveHandler)).Methods("POST")
	handler := cors.AllowAll().Handler(router)
	log.Fatal(http.ListenAndServeTLS(":1025", "/etc/letsencrypt/live/onryo.entrydns.org/cert.pem", "/etc/letsencrypt/live/onryo.entrydns.org/privkey.pem", handler))
}
