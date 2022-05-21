package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort = "8080"

	// TODO: Modify me: replace <script .. > ... </script> with your own, produced at https://core.telegram.org/widgets/login
	Html = `<!DOCTYPE html>
<html><head><title>Go Web Server</title></head>
<body><h1>Go Web Server</h1>
<h1>Hello, anonymous!</h1>
<script async src="https://telegram.org/js/telegram-widget.js?19" data-telegram-login="sgzmd_tgauth_bot" data-size="large" data-auth-url="http://tgauth.com/check_auth" data-request-access="write"></script>
</body></html>`
)

var (
	TgAuthKey string
)

const (
	TelegramCookie = "tg_auth"
	AuthUrl        = "/login"
	CheckAuthUrl   = "/check_auth"
)

// Checks if the user has successfully logged in with Telegram. It will return the
// json string of the user data if the user is logged in, otherwise it will return error.
func checkAuth(params map[string][]string) (map[string][]string, error) {
	// To check telegram login, we need to concat with "\n" all received fields _except_ hash
	// sorted in alphabetical order and then calculate hash using sha256, with the bot api key hash
	// as the secret.
	keys := make([]string, 0)
	for k := range params {
		if k != "hash" {
			keys = append(keys, k)
		}
	}

	dataCheckArray := make([]string, len(keys))
	for i, k := range keys {
		// e.g. username=the_user
		dataCheckArray[i] = k + "=" + params[k][0]
	}

	// strings in array should be sorted in alphabetical order
	sort.Strings(dataCheckArray)

	// producing string like id=8889999222&first_name=sgzmd&username=the_user&photo_url=https%3A%2F%2Ft.me%2Fi%2Fu...
	dataCheckStr := strings.Join(dataCheckArray, "\n")

	s256 := sha256.New()
	s256.Write([]byte(TgAuthKey))

	// We will now use this secret key to produce hash-based authentication code
	// from the dataCheckStr produced above
	secretKey := s256.Sum(nil)

	hm := hmac.New(sha256.New, secretKey)
	hm.Write([]byte(dataCheckStr))
	expectedHash := hex.EncodeToString(hm.Sum(nil))

	checkHash := params["hash"][0]

	// If the hashes match, then the request was indeed from Telegram
	if expectedHash != checkHash {
		return nil, fmt.Errorf("Hash mismatch")
	}

	// Now let's verify auth_date to check that the request is recent
	timestamp, err := strconv.ParseInt(params["auth_date"][0], 10, 64)
	if err != nil {
		return nil, err
	}

	// User must login every 24 hours
	if timestamp < (time.Now().Unix() - int64(24*time.Hour.Seconds())) {
		return nil, fmt.Errorf("User is not logged in for more than 24 hours")
	}

	return params, nil
}

// Checks if the user has successfully logged in with Telegram.
func checkAuthHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params := make(map[string][]string)
	for k, v := range r.Form {
		params[k] = v
	}

	p2, err := checkAuth(params)
	if err != nil {
		// if checkAuth returns error it means the parameters login page
		// has received are wrong, incorrect, or not from Telegram
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If we are here then auth has passed
	j, err := json.Marshal(p2)
	if err != nil {
		// This should practically never happen.
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Finally, let's set cookie so that we can check if the user is logged in later on.
	cookie := &http.Cookie{
		Name:    TelegramCookie,
		Expires: time.Now().Add(time.Hour * 24),
		Value:   url.QueryEscape(string(j)),
		Path:    "/",
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusFound)
}

func HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(Html))
}

func HandleIndexPage(w http.ResponseWriter, r *http.Request) {
	// We first check if the user is logged in
	cookie, err := r.Cookie(TelegramCookie)
	if err != nil {
		// If there's no login cookie, it guarantees that the user is not logged in.
		http.Redirect(w, r, AuthUrl, http.StatusFound)
		return
	}
	params := make(map[string][]string)
	data, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	e := json.Unmarshal([]byte(data), &params)
	if e != nil {
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := checkAuth(params); err != nil {
		// checkAuth returned error, which means user is not logged in - e.g. auth expired
		// or cookie doesn't look right.
		http.Redirect(w, r, AuthUrl, http.StatusFound)
		return
	}

	w.Write([]byte("<html><body><h1>Welcome, " + params["first_name"][0] + "</h1></body></html>"))
}

func main() {
	tgapi := flag.String("telegram_api_key", "", "Telegram API key")
	flag.Parse()

	if *tgapi == "" {
		panic("Telegram API key is required")
	}

	TgAuthKey = *tgapi

	http.HandleFunc(CheckAuthUrl, checkAuthHandler)
	http.HandleFunc(AuthUrl, HandleLoginPage)
	http.HandleFunc("/", HandleIndexPage)

	e := http.ListenAndServe("tgauth.com:"+defaultPort, nil)
	if e != nil {
		panic(e)
	}
}
