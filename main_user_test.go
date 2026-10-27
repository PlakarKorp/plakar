package main

import (
	"os"
	"strconv"
	"testing"
)

func TestUserFromEnv(t *testing.T) {
	t.Setenv("HOME", "/home/cron")

	// cron sets LOGNAME but not USER
	t.Setenv("LOGNAME", "aNickName")
	t.Setenv("USER", "")
	u, err := userFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "aNickName" || u.HomeDir != "/home/cron" {
		t.Fatalf("got %q at %q", u.Username, u.HomeDir)
	}

	t.Setenv("LOGNAME", "")
	t.Setenv("USER", "someone")
	if u, err = userFromEnv(); err != nil || u.Username != "someone" {
		t.Fatalf("got %+v, %v", u, err)
	}

	t.Setenv("USER", "")
	if u, err = userFromEnv(); err != nil || u.Username != strconv.Itoa(os.Getuid()) {
		t.Fatalf("got %+v, %v", u, err)
	}

	t.Setenv("HOME", "")
	if _, err = userFromEnv(); err == nil {
		t.Fatal("expected an error without $HOME")
	}
}
