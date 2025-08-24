package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetReleases(t *testing.T) {
	releases := Releases{
		{
			Version: "go1.25.0",
			Stable:  true,
			Files: []File{
				{
					Filename: "go1.25.0.linux-amd64.tar.gz",
					Os:       "linux",
					Arch:     "amd64",
					Version:  "go1.25.0",
					Sha256:   "2852af0c...",
					Size:     57 * 1024 * 1024,
					Kind:     "archive",
				},
			},
		},
	}

	t.Run("success", func(t *testing.T) {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(releases)
		}))

		defer srv.Close()

		got, err := GetReleases(srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 1 || got[0].Version != "go1.25.0" {
			t.Errorf("unexpected result: %+v", got)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))

		defer srv.Close()

		_, err := GetReleases(srv.Client(), srv.URL)
		if err == nil {
			t.Errorf("expected json decode error, got nil")
		}
	})

	t.Run("non 200 response", func(t *testing.T) {
		t.Helper()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad request", http.StatusBadRequest)
		}))

		defer srv.Close()

		_, err := GetReleases(srv.Client(), srv.URL)
		if err != nil {
			if errors.Is(err, errors.New("400 Bad Request")) {
				t.Errorf("expected 400 error, got %v", err)
			}
		}
	})

	t.Run("semantic verion check", func(t *testing.T) {
		t.Helper()

		want := "v1.25.0"

		got := withVPrefix("go1.25.0")
		if want != got {
			t.Fatalf("wanted %s, got %s", want, got)
		}

	})

}
