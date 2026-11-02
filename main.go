// Azeroth Survivors Save Editor: a local app that reads and writes the save and
// shows its interface in the browser. It listens on 127.0.0.1 only and every API
// call needs a random token generated at launch.
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed key.bin
var key []byte

//go:embed index.html
var indexHTML []byte

const magic = "RSE1:"

var (
	mu       sync.Mutex
	token    string
	lastPing = time.Now()
)

func keystream(nonce, data []byte) []byte {
	out := append([]byte(nil), data...)
	var block uint64
	for i := 0; i < len(out); i += 20 {
		m := hmac.New(sha1.New, key)
		m.Write([]byte("ks"))
		m.Write(nonce)
		var c [8]byte
		binary.LittleEndian.PutUint64(c[:], block)
		m.Write(c[:])
		ks := m.Sum(nil)
		for j := 0; j < 20 && i+j < len(out); j++ {
			out[i+j] ^= ks[j]
		}
		block++
	}
	return out
}

func tag(nonce, ct []byte) []byte {
	m := hmac.New(sha1.New, key)
	m.Write([]byte("tag"))
	m.Write(nonce)
	m.Write(ct)
	return m.Sum(nil)
}

func unseal(s []byte) ([]byte, error) {
	s = bytes.TrimSpace(s)
	if !bytes.HasPrefix(s, []byte(magic)) {
		return nil, errors.New("this is not an Azeroth Survivors save file")
	}
	b, err := base64.StdEncoding.DecodeString(string(s[len(magic):]))
	if err != nil || len(b) < 36 {
		return nil, errors.New("the save file is damaged")
	}
	n, ct, t := b[:16], b[16:len(b)-20], b[len(b)-20:]
	if !hmac.Equal(tag(n, ct), t) {
		return nil, errors.New("the save could not be verified. A game update may have changed its format, so this editor needs updating")
	}
	return keystream(n, ct), nil
}

func seal(p []byte) ([]byte, error) {
	n := make([]byte, 16)
	if _, err := rand.Read(n); err != nil {
		return nil, err
	}
	ct := keystream(n, p)
	raw := append(append(append([]byte(nil), n...), ct...), tag(n, ct)...)
	return []byte(magic + base64.StdEncoding.EncodeToString(raw)), nil
}

func savePath() (string, error) {
	if p := os.Getenv("WOW_ROGUE_SAVE"); p != "" {
		return p, nil
	}
	a := os.Getenv("APPDATA")
	if a == "" {
		return "", errors.New("could not find your AppData folder")
	}
	return filepath.Join(a, "wenilla-rogue", "characters.txt"), nil
}

func stampOf(p string) (string, error) {
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d-%d", st.Size(), st.ModTime().UnixNano()), nil
}

// writeAtomic writes to a temp file in the same folder, then renames it over the
// target, so a failure can never leave a half-written save.
func writeAtomic(p string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(p), ".editor-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func backupCurrent(p string) (string, error) {
	cur, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(p), "EditorBackups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	base := "characters-" + time.Now().Format("2006-01-02_15-04-05")
	name := base + ".txt"
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, name)); errors.Is(err, os.ErrNotExist) {
			break
		}
		name = fmt.Sprintf("%s-%d.txt", base, i)
	}
	if err := writeAtomic(filepath.Join(dir, name), cur); err != nil {
		return "", err
	}
	return name, nil
}

func replaceSave(p string, sealed []byte) (string, error) {
	want, err := unseal(sealed)
	if err != nil {
		return "", err
	}
	name, err := backupCurrent(p)
	if err != nil {
		return "", fmt.Errorf("the backup failed, so your save was not changed: %w", err)
	}
	if err := writeAtomic(p, sealed); err != nil {
		return "", fmt.Errorf("the save could not be written, and is unchanged: %w", err)
	}
	back, err := os.ReadFile(p)
	if err == nil {
		var got []byte
		if got, err = unseal(back); err == nil && !bytes.Equal(got, want) {
			err = errors.New("the data read back did not match")
		}
	}
	if err != nil {
		return "", fmt.Errorf("verification failed. Restore %s from the Backups tab: %w", name, err)
	}
	return name, nil
}

type backup struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Time   int64  `json:"time"`
	path   string
}

func listBackups(p string) []backup {
	root := filepath.Dir(p)
	out := []backup{}
	add := func(dir, source string, keep func(string) bool) {
		es, _ := os.ReadDir(dir)
		for _, e := range es {
			if e.IsDir() || !keep(e.Name()) {
				continue
			}
			fp := filepath.Join(dir, e.Name())
			f, err := os.Open(fp)
			if err != nil {
				continue
			}
			head := make([]byte, len(magic))
			_, err = io.ReadFull(f, head)
			f.Close()
			info, ierr := e.Info()
			if err != nil || ierr != nil || string(head) != magic {
				continue
			}
			out = append(out, backup{e.Name(), source, info.ModTime().Unix(), fp})
		}
	}
	all := func(string) bool { return true }
	add(filepath.Join(root, "EditorBackups"), "Editor backup", all)
	add(filepath.Join(root, "Saves"), "Game backup", func(n string) bool { return strings.HasSuffix(n, ".sav") })
	add(root, "Manual backup", func(n string) bool {
		return strings.HasPrefix(n, "characters.before-") && strings.HasSuffix(n, ".txt")
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, kind, msg string) {
	writeJSON(w, code, map[string]string{"error": msg, "kind": kind})
}

func api(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Token") != token {
			fail(w, 403, "auth", "Not allowed.")
			return
		}
		mu.Lock()
		defer mu.Unlock()
		lastPing = time.Now()
		h(w, r)
	}
}

func main() {
	tb := make([]byte, 24)
	if _, err := rand.Read(tb); err != nil {
		fatal("Could not start: " + err.Error())
	}
	token = hex.EncodeToString(tb)

	p, err := savePath()
	if err != nil {
		fatal(err.Error())
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal("Could not start: " + err.Error())
	}
	host := ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(indexHTML)
	})
	mux.HandleFunc("/api/load", api(func(w http.ResponseWriter, r *http.Request) {
		s, err := os.ReadFile(p)
		if err != nil {
			fail(w, 500, "read", "Could not read your save at "+p+". Has the game been run yet?")
			return
		}
		plain, err := unseal(s)
		if err != nil {
			fail(w, 500, "read", "Could not open your save: "+err.Error()+".")
			return
		}
		st, _ := stampOf(p)
		writeJSON(w, 200, map[string]any{"path": p, "text": string(plain), "stamp": st, "running": gameRunning()})
	}))
	mux.HandleFunc("/api/status", api(func(w http.ResponseWriter, r *http.Request) {
		st, _ := stampOf(p)
		writeJSON(w, 200, map[string]any{"stamp": st, "running": gameRunning()})
	}))
	mux.HandleFunc("/api/save", api(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Text, Stamp string }
		if r.Method != http.MethodPost || json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&q) != nil {
			fail(w, 400, "bad", "Bad request.")
			return
		}
		if gameRunning() {
			fail(w, 409, "running", "Close the game before saving.")
			return
		}
		if st, err := stampOf(p); err != nil || st != q.Stamp {
			fail(w, 409, "changed", "The save changed since it was opened, so nothing was written.")
			return
		}
		sealed, err := seal([]byte(q.Text))
		if err != nil {
			fail(w, 500, "write", err.Error())
			return
		}
		name, err := replaceSave(p, sealed)
		if err != nil {
			fail(w, 500, "write", err.Error())
			return
		}
		st, _ := stampOf(p)
		writeJSON(w, 200, map[string]any{"stamp": st, "backup": name})
	}))
	mux.HandleFunc("/api/backups", api(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, listBackups(p))
	}))
	mux.HandleFunc("/api/restore", api(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Name, Source string }
		if r.Method != http.MethodPost || json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&q) != nil {
			fail(w, 400, "bad", "Bad request.")
			return
		}
		if gameRunning() {
			fail(w, 409, "running", "Close the game before restoring.")
			return
		}
		// Only files from the listing can be restored; names are never used as paths.
		for _, b := range listBackups(p) {
			if b.Name == q.Name && b.Source == q.Source {
				data, err := os.ReadFile(b.path)
				if err == nil {
					_, err = unseal(data)
				}
				if err != nil {
					fail(w, 500, "write", "That backup could not be read: "+err.Error()+".")
					return
				}
				name, err := replaceSave(p, data)
				if err != nil {
					fail(w, 500, "write", err.Error())
					return
				}
				writeJSON(w, 200, map[string]any{"backup": name})
				return
			}
		}
		fail(w, 404, "missing", "That backup is no longer there.")
	}))

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != host {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			mux.ServeHTTP(w, r)
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	// The page pings every few seconds; quit a while after it is closed.
	go func() {
		for range time.Tick(15 * time.Second) {
			mu.Lock()
			idle := time.Since(lastPing)
			mu.Unlock()
			if idle > 3*time.Minute {
				os.Exit(0)
			}
		}
	}()
	url := "http://" + host + "/?t=" + token
	fmt.Println(url)
	openBrowser(url)
	srv.Serve(ln)
}
