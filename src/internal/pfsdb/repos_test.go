package pfsdb_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pachyderm/pachyderm/v2/src/pfs"

	"github.com/pachyderm/pachyderm/v2/src/internal/clusterstate"
	"github.com/pachyderm/pachyderm/v2/src/internal/dockertestenv"
	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/migrations"
	"github.com/pachyderm/pachyderm/v2/src/internal/pachsql"
	"github.com/pachyderm/pachyderm/v2/src/internal/pctx"
	"github.com/pachyderm/pachyderm/v2/src/internal/pfsdb"
	"github.com/pachyderm/pachyderm/v2/src/internal/protoutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/require"
	"github.com/pachyderm/pachyderm/v2/src/internal/testetcd"
	"github.com/pachyderm/pachyderm/v2/src/internal/testutil/random"
)

const (
	testRepoName = "testRepoName"
	testRepoType = "user"
	testRepoDesc = "this is a test repo"
)

func testRepo(name, repoType string) *pfs.RepoInfo {
	repo := &pfs.Repo{Name: name, Type: repoType, Project: &pfs.Project{Name: pfs.DefaultProjectName}}
	return &pfs.RepoInfo{
		Repo:        repo,
		Description: testRepoDesc,
	}
}

func compareRepos(t *testing.T, expected, got *pfs.RepoInfo) {
	t.Helper()
	expected = protoutil.Clone(expected)
	got = protoutil.Clone(got)
	expected.Created = nil
	got.Created = nil
	require.NoDiff(t, expected, got, []cmp.Option{protocmp.Transform()})
}

func newTestDB(t testing.TB, ctx context.Context) *pachsql.DB {
	db := dockertestenv.NewTestDB(t)
	migrationEnv := migrations.Env{EtcdClient: testetcd.NewEnv(ctx, t).EtcdClient}
	require.NoError(t, migrations.ApplyMigrations(ctx, db, migrationEnv, clusterstate.DesiredClusterState), "should be able to set up tables")
	return db
}

func TestUpsertRepo(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	expectedInfo := testRepo(testRepoName, testRepoType)
	var repoID pfsdb.RepoID
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		var err error
		repoID, err = pfsdb.UpsertRepo(ctx, tx, expectedInfo)
		require.NoError(t, err)
		getByIDInfo, err := pfsdb.GetRepoInfo(ctx, tx, repoID)
		require.NoError(t, err)
		compareRepos(t, expectedInfo, getByIDInfo)
		getByNameInfo, err := pfsdb.GetRepoByName(ctx, tx, expectedInfo.Repo.Project.Name, expectedInfo.Repo.Name, expectedInfo.Repo.Type)
		require.NoError(t, err)
		compareRepos(t, expectedInfo, getByNameInfo)
	})
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		expectedInfo.Description = "new desc"
		id, err := pfsdb.UpsertRepo(ctx, tx, expectedInfo)
		require.NoError(t, err)
		require.Equal(t, repoID, id, "UpsertRepo should keep id stable")
		getInfo, err := pfsdb.GetRepoInfo(ctx, tx, id)
		require.NoError(t, err)
		compareRepos(t, expectedInfo, getInfo)
	})
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		expectedInfo.Metadata = map[string]string{"key": "value"}
		id, err := pfsdb.UpsertRepo(ctx, tx, expectedInfo)
		require.NoError(t, err)
		require.Equal(t, repoID, id, "UpsertRepo should keep id stable")
		getInfo, err := pfsdb.GetRepoInfo(ctx, tx, id)
		require.NoError(t, err)
		compareRepos(t, expectedInfo, getInfo)
	})
}

func TestGetRepoByNameMissingProject(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	repo := testRepo(testRepoName, testRepoType)
	repo.Repo.Project.Name = "doesNotExist"
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		_, err := pfsdb.GetRepoByName(ctx, tx, repo.Repo.Project.Name, repo.Repo.Name, repo.Repo.Type)
		require.True(t, errors.As(err, &pfsdb.ProjectNotFoundError{}))
	})
}

func TestDeleteRepo(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	expectedInfo := testRepo(testRepoName, testRepoType)
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		id, err := pfsdb.UpsertRepo(ctx, tx, expectedInfo)
		require.NoError(t, err)
		require.NoError(t, pfsdb.DeleteRepo(ctx, tx, expectedInfo.Repo.Project.Name, expectedInfo.Repo.Name, expectedInfo.Repo.Type), "should be able to delete repo")
		_, err = pfsdb.GetRepoInfo(ctx, tx, id)
		require.True(t, errors.As(err, &pfsdb.RepoNotFoundError{}))
		require.True(t,
			errors.As(pfsdb.DeleteRepo(ctx, tx, expectedInfo.Repo.Project.Name, expectedInfo.Repo.Name, expectedInfo.Repo.Type),
				&pfsdb.RepoNotFoundError{Project: expectedInfo.Repo.Project.Name, Name: testRepoName, Type: expectedInfo.Repo.Type}),
		)
	})
}

func TestDeleteRepoMissingProject(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	expectedInfo := testRepo(testRepoName, testRepoType)
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		err := pfsdb.DeleteRepo(ctx, tx, "doesNotExist", expectedInfo.Repo.Name, expectedInfo.Repo.Type)
		require.True(t, errors.As(err, &pfsdb.ProjectNotFoundError{}))
	})
}

func createCommitAndBranches(ctx context.Context, tx *pachsql.Tx, t *testing.T, repoInfo *pfs.RepoInfo) {
	for _, branch := range repoInfo.Branches {
		commit := &pfs.Commit{Repo: repoInfo.Repo, Branch: nil, Id: random.String(32)}
		commitInfo := &pfs.CommitInfo{Commit: commit,
			Origin:  &pfs.CommitOrigin{Kind: pfs.OriginKind_USER},
			Started: timestamppb.Now()}
		commitID, err := pfsdb.CreateCommit(ctx, tx, commitInfo)
		require.NoError(t, err, "should be able to create commit")
		branchInfo := &pfs.BranchInfo{Branch: branch, Head: commit}
		branchID, err := pfsdb.UpsertBranch(ctx, tx, branchInfo)
		require.NoError(t, err, "should be able to create branch")
		commitInfo.Commit.Branch = branch
		err = pfsdb.UpdateCommitBranch(ctx, tx, commitID, branchID)
		require.NoError(t, err, "should be able to update commit")
	}
}

func TestGetRepo(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	createInfo := testRepo(testRepoName, testRepoType)
	createInfo.Branches = []*pfs.Branch{
		{Repo: createInfo.Repo, Name: "master"},
		{Repo: createInfo.Repo, Name: "a"},
		{Repo: createInfo.Repo, Name: "b"},
	}
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		repoID, err := pfsdb.UpsertRepo(ctx, tx, createInfo)
		require.NoError(t, err, "should be able to create repo")
		createCommitAndBranches(ctx, tx, t, createInfo)
		// validate GetRepoInfo.
		getInfo, err := pfsdb.GetRepoInfo(ctx, tx, repoID)
		require.NoError(t, err, "should be able to get a repo")
		compareRepos(t, createInfo, getInfo)
		// validate GetRepo.
		repo, err := pfsdb.GetRepo(ctx, tx, pfs.DefaultProjectName, testRepoName, testRepoType)
		require.NoError(t, err, "should be able to get a repo")
		compareRepos(t, createInfo, repo.RepoInfo)
		// validate error for attempting to get non-existent repo.
		_, err = pfsdb.GetRepoInfo(ctx, tx, 3)
		require.True(t, errors.As(err, &pfsdb.RepoNotFoundError{}))
	})
}

func TestForEachRepo(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		size := 210
		expectedInfos := make([]*pfs.RepoInfo, size)
		for i := 0; i < size; i++ {
			createInfo := testRepo(fmt.Sprintf("%s%03d", testRepoName, i), "unknown")
			id, err := pfsdb.UpsertRepo(ctx, tx, createInfo)
			require.NoError(t, err, "should be able to create repo")
			require.Equal(t, pfsdb.RepoID(i+1), id, "id should be auto incremented")
			createInfo.Branches = []*pfs.Branch{
				{Repo: createInfo.Repo, Name: "master"},
				{Repo: createInfo.Repo, Name: "a"},
				{Repo: createInfo.Repo, Name: "b"},
			}
			createCommitAndBranches(ctx, tx, t, createInfo)
			expectedInfos[i] = createInfo
		}
		i := 0
		require.NoError(t, pfsdb.ForEachRepo(ctx, tx, nil, nil, func(repo pfsdb.Repo) error {
			compareRepos(t, expectedInfos[i], repo.RepoInfo)
			i++
			return nil
		}))
		require.Equal(t, size, i)
	})
}

func TestForEachRepoFilter(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		for _, repoName := range []string{"repoA", "repoB", "repoC"} {
			for _, repoType := range []string{"user", "unknown", "meta"} {
				createInfo := testRepo(repoName, repoType)
				_, err := pfsdb.UpsertRepo(ctx, tx, createInfo)
				require.NoError(t, err, "should be able to create repo")
			}
		}
		filter := &pfsdb.RepoFilter{
			RepoTemplate: &pfs.Repo{Name: "repoA", Type: "meta", Project: &pfs.Project{Name: "default"}},
		}
		require.NoError(t, pfsdb.ForEachRepo(ctx, tx, filter, nil, func(repo pfsdb.Repo) error {
			require.Equal(t, "repoA", repo.RepoInfo.Repo.Name)
			require.Equal(t, "meta", repo.RepoInfo.Repo.Type)
			return nil
		}), "should be able to call for each repo")
		filter = &pfsdb.RepoFilter{
			RepoTemplate: &pfs.Repo{Name: "repoB", Type: "user", Project: &pfs.Project{Name: "default"}},
		}
		require.NoError(t, pfsdb.ForEachRepo(ctx, tx, filter, nil, func(repo pfsdb.Repo) error {
			require.Equal(t, "repoB", repo.RepoInfo.Repo.Name)
			require.Equal(t, "user", repo.RepoInfo.Repo.Type)
			return nil
		}), "should be able to call for each repo")
	})
}

func TestForEachRepoPage(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	collectRepoPage := func(tx *pachsql.Tx, page *pfs.RepoPage) []pfsdb.Repo {
		repos := make([]pfsdb.Repo, 0)
		require.NoError(t, pfsdb.ForEachRepo(ctx, tx, nil, page, func(repo pfsdb.Repo) error {
			repos = append(repos, repo)
			return nil
		}), "should be able to call for each repo")
		return repos
	}
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		for _, repoName := range []string{"A", "B", "C", "D", "E"} {
			createInfo := testRepo(repoName, "user")
			_, err := pfsdb.UpsertRepo(ctx, tx, createInfo)
			require.NoError(t, err, "should be able to create repo")
		}
		// page larger than number of repos
		rs := collectRepoPage(tx, &pfs.RepoPage{
			PageSize:  20,
			PageIndex: 0,
		})
		require.Equal(t, 5, len(rs))
		rs = collectRepoPage(tx, &pfs.RepoPage{
			PageSize:  3,
			PageIndex: 0,
		})
		assertRepoSequence(t, []string{"A", "B", "C"}, rs)
		rs = collectRepoPage(tx, &pfs.RepoPage{
			PageSize:  3,
			PageIndex: 1,
		})
		assertRepoSequence(t, []string{"D", "E"}, rs)
		// page overbounds
		rs = collectRepoPage(tx, &pfs.RepoPage{
			PageSize:  3,
			PageIndex: 2,
		})
		assertRepoSequence(t, []string{}, rs)
	})
}

// the default order for ForEachRepo is dictated by the page size and is (project.name, repo.name)
func TestForEachRepoDefaultOrder(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		project := &pfs.Project{Name: "alpha"}
		require.NoError(t,
			pfsdb.CreateProject(ctx, tx, &pfs.ProjectInfo{Project: project}),
			"should be able to create repo",
		)
		for i, repoName := range []string{"A", "B", "C"} {
			createInfo := testRepo(repoName, "user")
			if i == 2 {
				createInfo.Repo.Project.Name = "alpha"
			}
			_, err := pfsdb.UpsertRepo(ctx, tx, createInfo)
			require.NoError(t, err, "should be able to create repo")
		}
		repos := make([]pfsdb.Repo, 0)
		require.NoError(t, pfsdb.ForEachRepo(ctx, tx, nil, nil, func(repo pfsdb.Repo) error {
			repos = append(repos, repo)
			return nil
		}), "should be able to call for each repo")
		assertRepoSequence(t, []string{"C", "A", "B"}, repos)
	})
}

func TestForEachRepoPaginationWithFilter(t *testing.T) {
	t.Parallel()
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	listRepos := func(ctx context.Context, tx *pachsql.Tx, projects []string, page *pfs.RepoPage) []pfsdb.Repo {
		var projs []*pfs.Project
		for _, p := range projects {
			projs = append(projs, &pfs.Project{Name: p})
		}
		var repos []pfsdb.Repo
		filter := &pfsdb.RepoFilter{Projects: projs}
		require.NoError(t, pfsdb.ForEachRepo(ctx, tx, filter, page, func(repo pfsdb.Repo) error {
			repos = append(repos, repo)
			return nil
		}), "should be able to call for each repo")
		return repos
	}
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		project := &pfs.Project{Name: "alpha"}
		require.NoError(t,
			pfsdb.CreateProject(ctx, tx, &pfs.ProjectInfo{Project: project}),
			"should be able to create repo",
		)
		for i, repoName := range []string{"A", "B", "C", "D", "E", "F"} {
			createInfo := testRepo(repoName, "user")
			if i%2 == 0 {
				createInfo.Repo.Project.Name = "alpha"
			}
			_, err := pfsdb.UpsertRepo(ctx, tx, createInfo)
			require.NoError(t, err, "should be able to create repo")
		}
		assertRepoSequence(t,
			[]string{"B", "D"},
			listRepos(ctx,
				tx,
				[]string{pfs.DefaultProjectName},
				&pfs.RepoPage{PageSize: 2, PageIndex: 0},
			),
		)
		assertRepoSequence(t,
			[]string{"F"},
			listRepos(ctx,
				tx,
				[]string{pfs.DefaultProjectName},
				&pfs.RepoPage{PageSize: 2, PageIndex: 1},
			),
		)
	})
}

func assertRepoSequence(t *testing.T, names []string, repos []pfsdb.Repo) {
	require.Equal(t, len(names), len(repos))
	for i, n := range names {
		require.Equal(t, n, repos[i].RepoInfo.Repo.Name)
	}
}

func testRepoPicker() *pfs.RepoPicker {
	return &pfs.RepoPicker{
		Picker: &pfs.RepoPicker_Name{
			Name: &pfs.RepoPicker_RepoName{
				Project: testProjectPicker(),
				Name:    testRepoName,
				Type:    testRepoType,
			},
		},
	}
}

func TestPickRepo(t *testing.T) {
	t.Parallel()
	namePicker := testRepoPicker()
	badRepoPicker := proto.Clone(namePicker).(*pfs.RepoPicker)
	badRepoPicker.GetName().Name = "does not exist"
	repo := testRepo(testRepoName, testRepoType)
	ctx := pctx.TestContext(t)
	db := newTestDB(t, ctx)
	withTx(t, ctx, db, func(ctx context.Context, tx *pachsql.Tx) {
		_, err := pfsdb.UpsertRepo(ctx, tx, repo)
		require.NoError(t, err, "should be able to create repo")
		expected, err := pfsdb.GetRepo(ctx, tx, pfs.DefaultProjectName, testRepoName, testRepoType)
		require.NoError(t, err, "should be able to get a repo")
		got, err := pfsdb.PickRepo(ctx, namePicker, tx)
		require.NoError(t, err, "should be able to pick repo")
		require.Equal(t, expected.ID, got.ID)
		_, err = pfsdb.PickRepo(ctx, nil, tx)
		require.YesError(t, err, "pick repo should error with a nil picker")
		_, err = pfsdb.PickRepo(ctx, badRepoPicker, tx)
		require.YesError(t, err, "pick repo should error with bad picker")
		require.True(t, errors.As(err, &pfsdb.RepoNotFoundError{}))
	})
}
