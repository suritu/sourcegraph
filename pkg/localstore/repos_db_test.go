package localstore

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"context"

	"sourcegraph.com/sourcegraph/sourcegraph/pkg/accesscontrol"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/actor"
	sourcegraph "sourcegraph.com/sourcegraph/sourcegraph/pkg/api"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/github"
)

/*
 * Helpers
 */

func sortedRepoURIs(repos []*sourcegraph.Repo) []string {
	uris := repoURIs(repos)
	sort.Strings(uris)
	return uris
}

func repoURIs(repos []*sourcegraph.Repo) []string {
	var uris []string
	for _, repo := range repos {
		uris = append(uris, repo.URI)
	}
	return uris
}

func createRepo(ctx context.Context, t *testing.T, repo *sourcegraph.Repo) {
	if err := Repos.TryInsertNew(ctx, repo.URI, repo.Description, repo.Fork, repo.Private); err != nil {
		t.Fatal(err)
	}
}

func mustCreate(ctx context.Context, t *testing.T, repos ...*sourcegraph.Repo) []*sourcegraph.Repo {
	var createdRepos []*sourcegraph.Repo
	for _, repo := range repos {
		repo.DefaultBranch = "master"

		createRepo(ctx, t, repo)
		repo, err := Repos.GetByURI(ctx, repo.URI)
		if err != nil {
			t.Fatal(err)
		}
		createdRepos = append(createdRepos, repo)
	}
	return createdRepos
}

/*
 * Tests
 */

func TestRepos_Get(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	want := mustCreate(ctx, t, &sourcegraph.Repo{URI: "r"})

	repo, err := Repos.Get(ctx, want[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, repo, want[0]) {
		t.Errorf("got %v, want %v", repo, want[0])
	}
}

func TestRepos_List(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	ctx = actor.WithActor(ctx, &actor.Actor{})

	want := mustCreate(ctx, t, &sourcegraph.Repo{URI: "r"})

	repos, err := Repos.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, repos, want) {
		t.Errorf("got %v, want %v", repos, want)
	}
}

func TestRepos_List_pagination(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*sourcegraph.Repo{
		{URI: "r1"},
		{URI: "r2"},
		{URI: "r3"},
	}
	for _, repo := range createdRepos {
		mustCreate(ctx, t, repo)
	}

	type testcase struct {
		perPage int32
		page    int32
		exp     []string
	}
	tests := []testcase{
		{perPage: 1, page: 1, exp: []string{"r1"}},
		{perPage: 1, page: 2, exp: []string{"r2"}},
		{perPage: 1, page: 3, exp: []string{"r3"}},
		{perPage: 2, page: 1, exp: []string{"r1", "r2"}},
		{perPage: 2, page: 2, exp: []string{"r3"}},
		{perPage: 3, page: 1, exp: []string{"r1", "r2", "r3"}},
		{perPage: 3, page: 2, exp: nil},
		{perPage: 4, page: 1, exp: []string{"r1", "r2", "r3"}},
		{perPage: 4, page: 2, exp: nil},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, &RepoListOp{ListOptions: sourcegraph.ListOptions{PerPage: test.perPage, Page: test.page}})
		if err != nil {
			t.Fatal(err)
		}
		if got := sortedRepoURIs(repos); !reflect.DeepEqual(got, test.exp) {
			t.Errorf("for test case %v, got %v (want %v)", test, got, test.exp)
		}
	}
}

// TestRepos_List_query tests the behavior of Repos.List when called with
// a query.
// Test batch 1 (correct filtering)
func TestRepos_List_query1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*sourcegraph.Repo{
		{URI: "abc/def"},
		{URI: "def/ghi"},
		{URI: "jkl/mno/pqr"},
		{URI: "github.com/abc/xyz"},
	}
	for _, repo := range createdRepos {
		createRepo(ctx, t, repo)
	}
	tests := []struct {
		query string
		want  []string
	}{
		{"def", []string{"def/ghi", "abc/def"}},
		{"ABC/DEF", []string{"abc/def"}},
		{"xyz", []string{"github.com/abc/xyz"}},
		{"mno/p", []string{"jkl/mno/pqr"}},
		{"jkl mno pqr", []string{"jkl/mno/pqr"}},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, &RepoListOp{Query: test.query})
		if err != nil {
			t.Fatal(err)
		}
		if got := repoURIs(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%q: got repos %q, want %q", test.query, got, test.want)
		}
	}
}

// Test batch 2 (correct ranking)
func TestRepos_List_query2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	ctx = actor.WithActor(ctx, &actor.Actor{})

	createdRepos := []*sourcegraph.Repo{
		{URI: "a/def"},
		{URI: "b/def", Fork: true},
		{URI: "c/def", Private: true},
		{URI: "def/ghi"},
		{URI: "def/jkl", Fork: true},
		{URI: "def/mno", Private: true},
		{URI: "abc/m"},
	}
	for _, repo := range createdRepos {
		createRepo(ctx, t, repo)
	}
	tests := []struct {
		query string
		want  []string
	}{
		{"def", []string{"def/ghi", "def/jkl", "def/mno", "a/def", "b/def", "c/def"}},
		{"b/def", []string{"b/def"}},
		{"def/", []string{"def/ghi", "def/jkl", "def/mno"}},
		{"def/m", []string{"def/mno"}},
	}
	for _, test := range tests {
		repos, err := Repos.List(ctx, &RepoListOp{Query: test.query})
		if err != nil {
			t.Fatal(err)
		}
		if got := repoURIs(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%q: got repos %q, want %q", test.query, got, test.want)
		}
	}
}

func TestRepos_List_GitHub_Authenticated(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	calledListAccessible := github.MockListAccessibleRepos_Return([]*sourcegraph.Repo{
		&sourcegraph.Repo{URI: "github.com/is/privateButAccessible", Private: true, DefaultBranch: "master"},
	})
	github.GetRepoMock = func(ctx context.Context, uri string) (*sourcegraph.Repo, error) {
		if uri == "github.com/is/privateButAccessible" {
			return &sourcegraph.Repo{URI: "github.com/is/privateButAccessible", Private: true, DefaultBranch: "master"}, nil
		} else if uri == "github.com/is/public" {
			return &sourcegraph.Repo{URI: "github.com/is/public", Private: false, DefaultBranch: "master"}, nil
		}
		return nil, fmt.Errorf("unauthorized")
	}
	ctx = actor.WithActor(ctx, &actor.Actor{UID: "1", Login: "test", GitHubToken: "test"})

	createRepos := []*sourcegraph.Repo{
		&sourcegraph.Repo{URI: "a/local", Private: false, DefaultBranch: "master"},
		&sourcegraph.Repo{URI: "a/localPrivate", DefaultBranch: "master", Private: true},
		&sourcegraph.Repo{URI: "github.com/is/public", Private: false, DefaultBranch: "master"},
		&sourcegraph.Repo{URI: "github.com/is/privateButAccessible", Private: true, DefaultBranch: "master"},
		&sourcegraph.Repo{URI: "github.com/is/inaccessibleBecausePrivate", Private: true, DefaultBranch: "master"},
	}
	for _, repo := range createRepos {
		createRepo(ctx, t, repo)
	}

	ctx = accesscontrol.WithInsecureSkip(ctx, false) // use real access controls

	repoList, err := Repos.List(ctx, &RepoListOp{})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"a/local", "github.com/is/privateButAccessible", "github.com/is/public"}
	if got := sortedRepoURIs(repoList); !reflect.DeepEqual(got, want) {
		t.Fatalf("got repos %q, want %q", got, want)
	}
	if !*calledListAccessible {
		t.Error("!calledListAccessible")
	}
}

func TestRepos_List_GitHub_Authenticated_NoReposAccessible(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()
	ctx = accesscontrol.WithInsecureSkip(ctx, false) // use real access controls

	calledListAccessible := github.MockListAccessibleRepos_Return(nil)
	github.GetRepoMock = func(ctx context.Context, uri string) (*sourcegraph.Repo, error) {
		return nil, fmt.Errorf("unauthorized")
	}

	ctx = actor.WithActor(ctx, &actor.Actor{UID: "1", Login: "test", GitHubToken: "test"})

	createRepos := []*sourcegraph.Repo{
		&sourcegraph.Repo{URI: "github.com/not/accessible", DefaultBranch: "master", Private: true},
	}
	for _, repo := range createRepos {
		createRepo(ctx, t, repo)
	}

	repoList, err := Repos.List(ctx, &RepoListOp{})
	if err != nil {
		t.Fatal(err)
	}

	if len(repoList) != 0 {
		t.Errorf("got repos %v, want empty", repoList)
	}
	if !*calledListAccessible {
		t.Error("!calledListAccessible")
	}
}

func TestRepos_List_GitHub_Unauthenticated(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()
	ctx = accesscontrol.WithInsecureSkip(ctx, false) // use real access controls

	calledListAccessible := github.MockListAccessibleRepos_Return(nil)
	ctx = actor.WithActor(ctx, &actor.Actor{})
	github.GetRepoMock = func(ctx context.Context, uri string) (*sourcegraph.Repo, error) {
		return nil, fmt.Errorf("unauthorized")
	}

	createRepo(ctx, t, &sourcegraph.Repo{URI: "github.com/private", Private: true, DefaultBranch: "master"})

	repoList, err := Repos.List(ctx, &RepoListOp{})
	if err != nil {
		t.Fatal(err)
	}

	if got := sortedRepoURIs(repoList); len(got) != 0 {
		t.Fatal("List should not have returned any repos, got:", got)
	}

	if *calledListAccessible {
		t.Error("calledListAccessible, but wanted not called since there is no authed user")
	}
}

func TestRepos_Create(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	// Add a repo.
	createRepo(ctx, t, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"})

	repo, err := Repos.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if repo.CreatedAt == nil {
		t.Fatal("got CreatedAt nil")
	}
}

func TestRepos_Create_dupe(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := testContext()

	// Add a repo.
	createRepo(ctx, t, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"})

	// Add another repo with the same name.
	createRepo(ctx, t, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"})
}
