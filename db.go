package main

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func setActiveRepo(dtag string, repo string) error {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO activeRepo (dtag, repo) VALUES (?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(dtag, repo)
	if err != nil {
		return err
	}
	return nil
}

func getActiveRepo(user string) (string, bool) {
	var dtag, key string
	err := db.QueryRow("SELECT * FROM activeRepo WHERE dtag=?", user).Scan(&dtag, &key)
	if err != nil {
		return "", false
	}
	return key, true
}

func removeActiveRepo(user string) error {
	stmt, err := db.Prepare("DELETE FROM activeRepo WHERE dtag=?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(user)
	if err != nil {
		return err
	}
	return nil
}

func getGitlabUnameFromUser(id string) (string, error) {
	key, exists := associatedKey(id)
	if exists == false {
		return "", nil
	}
	user, ok := testKey(key)
	if ok == false {
		return "", nil
	}
	return user.Username, nil
}

func associatedKey(id string) (string, bool) {
	var dtag, key string
	err := db.QueryRow("SELECT * FROM gitlabKeys WHERE dtag=?", id).Scan(&dtag, &key)
	if err != nil {
		return "", false
	}
	return key, true
}

func testKey(key string) (gitlab.User, bool) {
	git := gitlab.NewClient(nil, key)
	user, _, err := git.Users.CurrentUser()
	if err != nil {
		fmt.Println(err)
		return gitlab.User{}, false
	}
	return *user, true
}

func associateUserToToken(user string, token string) error {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO gitlabKeys (dtag, key) VALUES (?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(user, token)
	if err != nil {
		return err
	}
	return nil
}
