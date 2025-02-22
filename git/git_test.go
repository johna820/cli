package git

import (
	"reflect"
	"testing"

	"github.com/cli/cli/v2/internal/run"
)

func TestLastCommit(t *testing.T) {
	t.Setenv("GIT_DIR", "./fixtures/simple.git")
	c, err := LastCommit()
	if err != nil {
		t.Fatalf("LastCommit error: %v", err)
	}
	if c.Sha != "6f1a2405cace1633d89a79c74c65f22fe78f9659" {
		t.Errorf("expected sha %q, got %q", "6f1a2405cace1633d89a79c74c65f22fe78f9659", c.Sha)
	}
	if c.Title != "Second commit" {
		t.Errorf("expected title %q, got %q", "Second commit", c.Title)
	}
}

func TestCommitBody(t *testing.T) {
	t.Setenv("GIT_DIR", "./fixtures/simple.git")
	body, err := CommitBody("6f1a2405cace1633d89a79c74c65f22fe78f9659")
	if err != nil {
		t.Fatalf("CommitBody error: %v", err)
	}
	if body != "I'm starting to get the hang of things\n" {
		t.Errorf("expected %q, got %q", "I'm starting to get the hang of things\n", body)
	}
}

/*
	NOTE: below this are stubbed git tests, i.e. those that do not actually invoke `git`. If possible, utilize
	`setGitDir()` to allow new tests to interact with `git`. For write operations, you can use `t.TempDir()` to
	host a temporary git repository that is safe to be changed.
*/

func Test_UncommittedChangeCount(t *testing.T) {
	type c struct {
		Label    string
		Expected int
		Output   string
	}
	cases := []c{
		{Label: "no changes", Expected: 0, Output: ""},
		{Label: "one change", Expected: 1, Output: " M poem.txt"},
		{Label: "untracked file", Expected: 2, Output: " M poem.txt\n?? new.txt"},
	}

	for _, v := range cases {
		t.Run(v.Label, func(t *testing.T) {
			cs, restore := run.Stub()
			defer restore(t)
			cs.Register(`git status --porcelain`, 0, v.Output)

			ucc, _ := UncommittedChangeCount()
			if ucc != v.Expected {
				t.Errorf("UncommittedChangeCount() = %d, expected %d", ucc, v.Expected)
			}
		})
	}
}

func Test_CurrentBranch(t *testing.T) {
	type c struct {
		Stub     string
		Expected string
	}
	cases := []c{
		{
			Stub:     "branch-name\n",
			Expected: "branch-name",
		},
		{
			Stub:     "refs/heads/branch-name\n",
			Expected: "branch-name",
		},
		{
			Stub:     "refs/heads/branch\u00A0with\u00A0non\u00A0breaking\u00A0space\n",
			Expected: "branch\u00A0with\u00A0non\u00A0breaking\u00A0space",
		},
	}

	for _, v := range cases {
		cs, teardown := run.Stub()
		cs.Register(`git symbolic-ref --quiet HEAD`, 0, v.Stub)

		result, err := CurrentBranch()
		if err != nil {
			t.Errorf("got unexpected error: %v", err)
		}
		if result != v.Expected {
			t.Errorf("unexpected branch name: %s instead of %s", result, v.Expected)
		}
		teardown(t)
	}
}

func Test_CurrentBranch_detached_head(t *testing.T) {
	cs, teardown := run.Stub()
	defer teardown(t)
	cs.Register(`git symbolic-ref --quiet HEAD`, 1, "")

	_, err := CurrentBranch()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err != ErrNotOnAnyBranch {
		t.Errorf("got unexpected error: %s instead of %s", err, ErrNotOnAnyBranch)
	}
}

func TestParseExtraCloneArgs(t *testing.T) {
	type Wanted struct {
		args []string
		dir  string
	}
	tests := []struct {
		name string
		args []string
		want Wanted
	}{
		{
			name: "args and target",
			args: []string{"target_directory", "-o", "upstream", "--depth", "1"},
			want: Wanted{
				args: []string{"-o", "upstream", "--depth", "1"},
				dir:  "target_directory",
			},
		},
		{
			name: "only args",
			args: []string{"-o", "upstream", "--depth", "1"},
			want: Wanted{
				args: []string{"-o", "upstream", "--depth", "1"},
				dir:  "",
			},
		},
		{
			name: "only target",
			args: []string{"target_directory"},
			want: Wanted{
				args: []string{},
				dir:  "target_directory",
			},
		},
		{
			name: "no args",
			args: []string{},
			want: Wanted{
				args: []string{},
				dir:  "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, dir := parseCloneArgs(tt.args)
			got := Wanted{
				args: args,
				dir:  dir,
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestAddNamedRemote(t *testing.T) {
	tests := []struct {
		title    string
		name     string
		url      string
		dir      string
		branches []string
		want     string
	}{
		{
			title:    "fetch all",
			name:     "test",
			url:      "URL",
			dir:      "DIRECTORY",
			branches: []string{},
			want:     "git -C DIRECTORY remote add -f test URL",
		},
		{
			title:    "fetch specific branches only",
			name:     "test",
			url:      "URL",
			dir:      "DIRECTORY",
			branches: []string{"trunk", "dev"},
			want:     "git -C DIRECTORY remote add -t trunk -t dev -f test URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			cs, cmdTeardown := run.Stub()
			defer cmdTeardown(t)

			cs.Register(tt.want, 0, "")

			err := AddNamedRemote(tt.url, tt.name, tt.dir, tt.branches)
			if err != nil {
				t.Fatalf("error running command `git remote add -f`: %v", err)
			}
		})
	}
}
