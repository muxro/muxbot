package main

import "testing"

import "strings"

import "github.com/xanzy/go-gitlab"

// TODO

func initDummyGit() {

}

var parserAddTests = []struct {
	in      string
	outData IssuesAddOptions
	err     error
}{
	{"Name -- test", IssuesAddOptions{Title: "Name", Description: "test"}, nil},
}

func areAddOptionsSame(a, b IssuesAddOptions) bool {
	if a.Assignee != b.Assignee ||
		a.Title != b.Title ||
		a.Description != b.Description ||
		!arrayEqual(a.Tags, b.Tags) ||
		a.ProjectID != b.ProjectID {
		return false
	}
	return true
}

func arrayEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseAddOpts(t *testing.T) {
	for _, tt := range parserAddTests {
		t.Run(tt.in, func(t *testing.T) {
			args := strings.Split(tt.in, " ")
			projects := []*gitlab.Project{}
			git := &gitlab.Client{}
			opts, err := parseAddOpts(args, projects, git)
			if !areAddOptionsSame(opts, tt.outData) && err != tt.err {
				t.Errorf("got %#v %#v, want %#v %#v\n", opts, err, tt.outData, tt.err)
			}
		})
	}
}
