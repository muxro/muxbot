package main

import (
	"reflect"
	"strings"
	"testing"
)

var parserAddTests = []struct {
	in      string
	outData IssuesAddOptions
	err     error
}{
	{"Name -- test", IssuesAddOptions{Title: "Name", Description: "test"}, nil},
	{"$CombineNConquer +Critical test -- This", IssuesAddOptions{Assignee: "CombineNConquer", Tags: []string{"Critical"}, Title: "test", Description: "This"}, nil},
	{"$CombineNConquer +Critical", IssuesAddOptions{}, errNoTitleSpecified},
}

func TestParseAddOpts(t *testing.T) {
	for _, tt := range parserAddTests {
		t.Run(tt.in, func(t *testing.T) {
			args := strings.Split(tt.in, " ")
			opts, err := parseAddOpts(args)
			if reflect.DeepEqual(opts, tt.outData) == false || err != tt.err {
				t.Errorf("got %#v %#v, want %#v %#v\n", opts, err, tt.outData, tt.err)
			}
		})
	}
}

func TestParseModifyOpts(t *testing.T) {

}

func TestParseListOpts(t *testing.T) {

}

func TestParseIssueParamOpts(t *testing.T) {

}
