package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
)

// session is a session info of logged in user.
type session struct {
	User    string
	Session string
}

func main() {
	exec := os.Args[0]
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println(exec + ": need forge url")
		return
	}
	host := args[0]
	// check existing session is valid
	b, err := readConfigFile("session")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("read session: %v", err)
		}
	}
	if b != nil {
		session := string(b)
		ok, err := testSession(host, string(session))
		if err != nil {
			log.Fatalf("test session: %v", err)
		}
		if ok {
			fmt.Println(session)
			return
		}
	}
	// don't have valid session, get a new session
	key, err := generateRandomString(64)
	if err != nil {
		log.Fatalf("generate random string: %v", err)
	}
	err = openForgeLoginPage(host, key)
	if err != nil {
		log.Fatalf("open login page: %v", err)
	}
	s, err := getSession(host, key)
	if err != nil {
		log.Fatalf("get session info: %v", err)
	}
	data := []byte(s.Session)
	err = writeConfigFile("session", data)
	if err != nil {
		log.Fatalf("write session: %v", err)
	}
	fmt.Println(s.Session)
}

// generateRandomString returns random string that has length 'n', using alpha-numeric characters.
func generateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}

// openForgeLoginPage shows login page to user.
func openForgeLoginPage(host, key string) error {
	return openPath("https://" + host + "/login?app_session_key=" + key)
}

// openPath opens a path which can be a file, directory, or url.
func openPath(path string) error {
	var open []string
	switch runtime.GOOS {
	case "windows":
		open = []string{"cmd", "/c", "start " + path}
	case "darwin":
		open = []string{"open", path}
	case "linux":
		open = []string{"xdg-open", path}
	default:
		log.Fatalf("unsupported os: %s", runtime.GOOS)
	}
	cmd := exec.Command(open[0], open[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

// apiResponse is form of forge api response.
type apiResponse struct {
	Msg interface{}
	Err string
}

// decodeAPIResponse decodes api response into dest.
func decodeAPIResponse(resp *http.Response, dest interface{}) error {
	r := apiResponse{Msg: dest}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &r)
	if err != nil {
		return fmt.Errorf("%s: %s", err, b)
	}
	if r.Err != "" {
		return fmt.Errorf(r.Err)
	}
	return nil
}

// getSession gets session from the forge host.
func getSession(host, key string) (session, error) {
	resp, err := http.PostForm("https://"+host+"/api/app-login", url.Values{
		"key": {key},
	})
	if err != nil {
		return session{}, err
	}
	var s session
	err = decodeAPIResponse(resp, &s)
	if err != nil {
		return session{}, err
	}
	return s, nil
}

// testSession tests the session from the forge host.
func testSession(host, session string) (bool, error) {
	resp, err := http.PostForm("https://"+host+"/api/test-session", url.Values{
		"session": {string(session)},
	})
	if err != nil {
		return false, err
	}
	var ok bool
	err = decodeAPIResponse(resp, &ok)
	if err != nil {
		// assume this is a invalid session.
		return false, nil
	}
	return ok, nil
}

// readConfigFile writes data to a config file.
func readConfigFile(filename string) ([]byte, error) {
	confd, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(confd + "/forge/" + filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// writeConfigFile writes data to a config file.
func writeConfigFile(filename string, data []byte) error {
	confd, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	err = os.MkdirAll(confd+"/forge", 0755)
	if err != nil {
		return err
	}
	f, err := os.Create(confd + "/forge/" + filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}
