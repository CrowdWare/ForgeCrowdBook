package model

import (
	"database/sql"
	"path/filepath"
	"testing"

	"codeberg.org/crowdware/forgecrowdbook/internal/db"
)

func TestGetUserByEmailNotFoundReturnsNil(t *testing.T) {
	database := openModelTestDB(t)

	user, err := GetUserByEmail(database, "missing@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if user != nil {
		t.Fatalf("expected nil user, got %+v", user)
	}
}

func TestCreateUserDuplicateEmailFails(t *testing.T) {
	database := openModelTestDB(t)

	if _, err := CreateUser(database, "dup@example.com", "One"); err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}

	if _, err := CreateUser(database, "dup@example.com", "Two"); err == nil {
		t.Fatal("expected duplicate email error")
	}
}

func TestUpdateLangAndBanUser(t *testing.T) {
	database := openModelTestDB(t)

	created, err := CreateUser(database, "user@example.com", "User")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if err := UpdateUserLang(database, created.ID, "de"); err != nil {
		t.Fatalf("UpdateUserLang failed: %v", err)
	}
	if err := BanUser(database, created.ID); err != nil {
		t.Fatalf("BanUser failed: %v", err)
	}

	loaded, err := GetUserByID(database, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected user to exist")
	}
	if loaded.Lang != "de" {
		t.Fatalf("expected lang de, got %q", loaded.Lang)
	}
	if loaded.Status != "banned" {
		t.Fatalf("expected status banned, got %q", loaded.Status)
	}
}

func openModelTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "model.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}
