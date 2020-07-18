package testutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	testing "github.com/mitchellh/go-testing-interface"
)

var root string

// RunAnnictServer runs a dummy Annict server for testing and returns the server address.
func RunAnnictServer(t testing.T, codeDecider map[string]int) (addr string) {
	var buf bytes.Buffer
	cmd := exec.Command("git", "rev-parse", "--show-cdup")
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	root = strings.TrimSpace(buf.String())

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		s := string(b)

		for name, code := range codeDecider {
			if strings.Contains(s, name) {
				w.WriteHeader(code)
				return
			}
		}

		switch {
		case strings.Contains(s, "GetWork"):
			copyFile(t, w, "get_work_response")
		case strings.Contains(s, "GetProfile"):
			copyFile(t, w, "get_profile_response")
		case strings.Contains(s, "ListWorks"):
			copyFile(t, w, "list_works_response")
		case strings.Contains(s, "listRecords"):
			copyFile(t, w, "list_records_response")
		case strings.Contains(s, "ListNextEpisodes"):
			copyFile(t, w, "list_next_episodes_response")
		case strings.Contains(s, "CreateRecordMutation"),
			strings.Contains(s, "UpdateStatusMutation"):
			if _, err := io.WriteString(w, `{"data": {}}`); err != nil {
				t.Fatal(err)
			}
		default:
			t.Error("unknown query")
		}
	})

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func copyFile(t testing.T, w io.Writer, fname string) {
	f, err := os.Open(filepath.Join(root, "testutil", "testdata", fname))
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		t.Error(err)
		return
	}
}
