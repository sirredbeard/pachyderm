package pfsdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jmoiron/sqlx"
	"github.com/pachyderm/pachyderm/v2/src/internal/authdb"
	"github.com/pachyderm/pachyderm/v2/src/internal/collection"
	"github.com/pachyderm/pachyderm/v2/src/internal/dbutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/pachsql"
	"github.com/pachyderm/pachyderm/v2/src/internal/pgjsontypes"
	"github.com/pachyderm/pachyderm/v2/src/internal/randutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/stream"
	"github.com/pachyderm/pachyderm/v2/src/internal/uuid"
	"github.com/pachyderm/pachyderm/v2/src/internal/watch/postgres"
	"github.com/pachyderm/pachyderm/v2/src/pfs"
)

const (
	BranchesChannelName     = "pfs_branches"
	BranchesRepoChannelName = "pfs_branches_repo_"
)

const (
	getBranchBaseQuery = `
		SELECT
			branch.id,
			branch.name,
			branch.created_at,
			branch.updated_at,
			branch.metadata,
			branch.created_by,
			repo.id as "repo.id",
			repo.name as "repo.name",
			repo.type as "repo.type",
			project.id as "repo.project.id",
			project.name as "repo.project.name",
			commit.commit_id as "head.commit_id",
			commit.commit_set_id as "head.commit_set_id",
			headBranch.name as "head.branch_name",
			repo.name as "head.repo.name",
			repo.type as "head.repo.type",
			project.name as "head.repo.project.name"
		FROM pfs.branches branch
			JOIN pfs.repos repo ON branch.repo_id = repo.id
			JOIN core.projects project ON repo.project_id = project.id
			JOIN pfs.commits commit ON branch.head = commit.int_id
			LEFT JOIN pfs.branches headBranch on commit.branch_id = headBranch.id
	`
	getBranchByIDQuery   = getBranchBaseQuery + ` WHERE branch.id = $1`
	getBranchByNameQuery = getBranchBaseQuery + ` WHERE project.name = $1 AND repo.name = $2 AND repo.type = $3 AND branch.name = $4`
	branchesPageSize     = 100
)

type branchColumn string

const (
	BranchColumnID        = branchColumn("branch.id")
	BranchColumnRepoID    = branchColumn("branch.repo_id")
	BranchColumnCreatedAt = branchColumn("branch.created_at")
	BranchColumnUpdatedAt = branchColumn("branch.updated_at")
)

// Ensures BranchIterator implements the Iterator interface.
var _ stream.Iterator[Branch] = &BranchIterator{}

// BranchProvCycleError is returned when a cycle is detected at branch creation time.
type BranchProvCycleError struct {
	From, To string
}

func (err *BranchProvCycleError) Error() string {
	return fmt.Sprintf("cycle detected because %v is already in the subvenance of %v", err.To, err.From)
}

func (err *BranchProvCycleError) GRPCStatus() *status.Status {
	return status.New(codes.Internal, err.Error())
}

// BranchNotFoundError is returned when a branch is not found in postgres.
type BranchNotFoundError struct {
	ID        BranchID
	BranchKey string
}

func (err *BranchNotFoundError) Error() string {
	if strings.Contains(err.BranchKey, pfs.UserRepoType) {
		branchKeyWithoutUser := strings.Replace(err.BranchKey, "."+pfs.UserRepoType, "", 1)
		return fmt.Sprintf("branch (id=%d, branch=%s) not found", err.ID, branchKeyWithoutUser)
	}
	return fmt.Sprintf("branch (id=%d, branch=%s) not found", err.ID, err.BranchKey)
}

func (err *BranchNotFoundError) GRPCStatus() *status.Status {
	return status.New(codes.NotFound, err.Error())
}

// BranchesInRepoChannel returns the name of the channel that is notified when branches in repo 'repoID' are created, updated, or deleted
func BranchesInRepoChannel(repoID RepoID) string {
	return fmt.Sprintf("%s%d", BranchesRepoChannelName, repoID)
}

type BranchIterator struct {
	paginator pageIterator[BranchRow]
	ext       sqlx.ExtContext
}

// Branch wraps a *pfs.BranchInfo with an ID and an optional Revision.
// The Revision is set by a BranchIterator.
type Branch struct {
	ID       BranchID
	Revision int64
	*pfs.BranchInfo
}

type OrderByBranchColumn OrderByColumn[branchColumn]

func NewBranchIterator(ctx context.Context, ext sqlx.ExtContext, startPage, pageSize uint64, filter *pfs.Branch, orderBys ...OrderByBranchColumn) (*BranchIterator, error) {
	var conditions []string
	var values []any
	// Note that using ? as the bindvar is okay because we rebind it later.
	if filter != nil {
		if filter.Repo.Project != nil && filter.Repo.Project.Name != "" {
			conditions = append(conditions, "project.name = ?")
			values = append(values, filter.Repo.Project.Name)
		}
		if filter.Repo != nil && filter.Repo.Name != "" {
			conditions = append(conditions, "repo.name = ?")
			values = append(values, filter.Repo.Name)
		}
		if filter.Repo != nil && filter.Repo.Type != "" {
			conditions = append(conditions, "repo.type = ?")
			values = append(values, filter.Repo.Type)
		}
		if filter.Name != "" {
			conditions = append(conditions, "branch.name = ?")
			values = append(values, filter.Name)
		}
	}
	query := getBranchBaseQuery
	if len(conditions) > 0 {
		query += "\n" + fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}
	// Default ordering is by branch id in ascending order. This is important for pagination.
	var orderByGeneric []OrderByColumn[branchColumn]
	if len(orderBys) == 0 {
		orderByGeneric = []OrderByColumn[branchColumn]{{Column: BranchColumnID, Order: SortOrderAsc}}
	} else {
		for _, orderBy := range orderBys {
			orderByGeneric = append(orderByGeneric, OrderByColumn[branchColumn](orderBy))
		}
	}
	query += "\n" + OrderByQuery[branchColumn](orderByGeneric...)
	query = ext.Rebind(query)
	return &BranchIterator{
		paginator: newPageIterator[BranchRow](ctx, query, values, startPage, pageSize, 0),
		ext:       ext,
	}, nil
}

func ForEachBranch(ctx context.Context, tx *pachsql.Tx, filter *pfs.Branch, cb func(branch Branch) error, orderBys ...OrderByBranchColumn) error {
	iter, err := NewBranchIterator(ctx, tx, 0, 100, filter, orderBys...)
	if err != nil {
		return errors.Wrap(err, "for each branch")
	}
	if err := stream.ForEach[Branch](ctx, iter, cb); err != nil {
		return errors.Wrap(err, "for each branch")
	}
	return nil
}

func ListBranches(ctx context.Context, tx *pachsql.Tx, filter *pfs.Branch, orderBys ...OrderByBranchColumn) ([]Branch, error) {
	var branches []Branch
	err := ForEachBranch(ctx, tx, filter, func(branch Branch) error {
		branches = append(branches, branch)
		return nil
	}, orderBys...)
	if err != nil {
		return nil, errors.Wrap(err, "list branches")
	}
	return branches, nil
}

func (i *BranchIterator) Next(ctx context.Context, dst *Branch) error {
	if dst == nil {
		return errors.Errorf("dst BranchInfo cannot be nil")
	}
	branch, rev, err := i.paginator.next(ctx, i.ext)
	if err != nil {
		return err
	}
	branchInfo, err := fetchBranchInfoByBranch(ctx, i.ext, branch)
	if err != nil {
		return err
	}
	dst.ID = branch.ID
	dst.BranchInfo = branchInfo
	dst.Revision = rev
	return nil
}

// GetBranchInfo returns a *pfs.BranchInfo by id.
func GetBranchInfo(ctx context.Context, tx *pachsql.Tx, id BranchID) (*pfs.BranchInfo, error) {
	branch := &BranchRow{}
	if err := tx.GetContext(ctx, branch, getBranchByIDQuery, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, &BranchNotFoundError{ID: id}
		}
		return nil, errors.Wrap(err, "could not get branch")
	}
	return fetchBranchInfoByBranch(ctx, tx, branch)
}

// GetBranch returns a *pfsdb.Branch by name.
func GetBranch(ctx context.Context, tx *pachsql.Tx, b *pfs.Branch) (*Branch, error) {
	if b == nil {
		return nil, errors.Errorf("branch cannot be nil")
	}
	row := &BranchRow{}
	project := b.GetRepo().GetProject().GetName()
	repo := b.GetRepo().GetName()
	repoType := b.GetRepo().GetType()
	branch := b.GetName()
	if err := tx.GetContext(ctx, row, getBranchByNameQuery, project, repo, repoType, branch); err != nil {
		if err == sql.ErrNoRows {
			if _, err := GetRepoByName(ctx, tx, project, repo, repoType); err != nil {
				if errors.As(err, new(*RepoNotFoundError)) {
					return nil, errors.Join(err, &BranchNotFoundError{BranchKey: b.Key()})
				}
				return nil, errors.Wrapf(err, "get repo for branch info %v", b.Key())
			}
			return nil, &BranchNotFoundError{BranchKey: b.Key()}
		}
		return nil, errors.Wrap(err, "could not get branch")
	}
	branchInfo, err := fetchBranchInfoByBranch(ctx, tx, row)
	if err != nil {
		return nil, err
	}
	return &Branch{ID: row.ID, BranchInfo: branchInfo}, nil
}

// GetBranchID returns the id of a branch given a set strings that uniquely identify a branch.
func GetBranchID(ctx context.Context, tx *pachsql.Tx, branch *pfs.Branch) (BranchID, error) {
	var id BranchID
	if err := tx.GetContext(ctx, &id, `
		SELECT branch.id
		FROM pfs.branches branch
			JOIN pfs.repos repo ON branch.repo_id = repo.id
			JOIN core.projects project ON repo.project_id = project.id
		WHERE project.name = $1 AND repo.name = $2 AND repo.type = $3 AND branch.name = $4
	`,
		branch.Repo.Project.Name,
		branch.Repo.Name,
		branch.Repo.Type,
		branch.Name,
	); err != nil {
		if err == sql.ErrNoRows {
			return 0, &BranchNotFoundError{BranchKey: branch.Key()}
		}
		return 0, errors.Wrapf(err, "could not get id for branch %s", branch.Key())
	}
	return id, nil
}

// UpsertBranch creates a branch if it does not exist, or updates the head if the branch already exists.
// If direct provenance is specified, it will be used to update the branch's provenance relationships.
func UpsertBranch(ctx context.Context, tx *pachsql.Tx, branchInfo *pfs.BranchInfo) (BranchID, error) {
	if branchInfo.Branch.Repo.Name == "" {
		return 0, errors.Errorf("repo name required")
	}
	if branchInfo.Branch.Repo.Type == "" {
		return 0, errors.Errorf("repo type required")
	}
	if branchInfo.Branch.Repo.Project.Name == "" {
		return 0, errors.Errorf("project name required")
	}
	if branchInfo.Head.Id == "" {
		return 0, errors.Errorf("head commit required")
	}
	if uuid.IsUUIDWithoutDashes(branchInfo.Branch.Name) {
		return 0, errors.Errorf("branch name cannot be a UUID V4")
	}
	var createdBy sql.NullString
	if cb := branchInfo.CreatedBy; cb != "" {
		createdBy.String = cb
		createdBy.Valid = true
		if err := authdb.EnsurePrincipal(ctx, tx, cb); err != nil {
			return 0, errors.Wrapf(err, "ensure principal %v", cb)
		}
	}
	var branchID BranchID
	// TODO stop matching on pfs.commits.commit_id, because that will eventually be deprecated.
	// Instead, construct the commit_id based on existing project, repo, and commit_set_id fields.
	//
	// Note: on conflict, we don't touch created_by.
	if err := tx.QueryRowContext(ctx,
		`
		INSERT INTO pfs.branches(repo_id, name, head, metadata, created_by)
		VALUES (
			(SELECT repo.id FROM pfs.repos repo JOIN core.projects project ON repo.project_id = project.id WHERE project.name = $1 AND repo.name = $2 AND repo.type = $3),
			$4,
			(SELECT int_id FROM pfs.commits WHERE commit_id = $5),
			$6,
			$7
		)
		ON CONFLICT (repo_id, name) DO UPDATE SET
			head = EXCLUDED.head,
			metadata = EXCLUDED.metadata
		RETURNING id
		`,
		branchInfo.Branch.Repo.Project.Name,
		branchInfo.Branch.Repo.Name,
		branchInfo.Branch.Repo.Type,
		branchInfo.Branch.Name,
		CommitKey(branchInfo.Head),
		pgjsontypes.StringMap{Data: branchInfo.Metadata},
		createdBy,
	).Scan(&branchID); err != nil {
		return 0, errors.Wrap(err, "could not create branch")
	}
	// Compute branch provenance, and avoid creating cycles.
	// We know a cycle exists if the to_branch is in the subvenance of the from_branch.
	// Note that we get the full subvenance set as an efficiency optimization,
	// where we avoid having to query the database for each branch in the provenance chain.
	fullSubv, err := GetFullBranchSubvenance(ctx, tx, branchID)
	if err != nil {
		return branchID, errors.Wrap(err, "could not compute branch subvenance")
	}
	fullSubvSet := make(map[string]bool)
	for _, branch := range fullSubv {
		fullSubvSet[branch.Key()] = true
	}
	for _, toBranch := range branchInfo.DirectProvenance {
		if fullSubvSet[toBranch.Key()] {
			return branchID, &BranchProvCycleError{From: branchInfo.Branch.Key(), To: toBranch.Key()}
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pfs.branch_provenance WHERE from_id = $1`, branchID); err != nil {
		return branchID, errors.Wrap(err, "could not delete direct branch provenance")
	}
	for _, branch := range branchInfo.DirectProvenance {
		toBranchID, err := GetBranchID(ctx, tx, branch)
		if err != nil {
			return branchID, errors.Wrapf(err, "could not get to_branch_id for creating branch provenance")
		}
		if err := CreateDirectBranchProvenance(ctx, tx, branchID, toBranchID); err != nil {
			return branchID, errors.Wrap(err, "could not create branch provenance")
		}
	}
	// Create or update this branch's trigger.
	if branchInfo.Trigger != nil {
		toBranchID, err := GetBranchID(ctx, tx, &pfs.Branch{Repo: branchInfo.Branch.Repo, Name: branchInfo.Trigger.Branch})
		if err != nil {
			return branchID, errors.Wrap(err, "updating branch trigger")
		}
		if err := UpsertBranchTrigger(ctx, tx, branchID, toBranchID, branchInfo.Trigger); err != nil {
			return branchID, errors.Wrap(err, "updating branch trigger")
		}
	} else {
		// Delete existing branch trigger.
		if err := DeleteBranchTrigger(ctx, tx, branchID); err != nil {
			return branchID, errors.Wrap(err, "updating branch trigger")
		}
	}
	// Update the branch propagation specs.
	for _, bps := range branchInfo.BranchPropagationSpecs {
		toBranchID, err := GetBranchID(ctx, tx, bps.Branch)
		if err != nil {
			return branchID, errors.Wrapf(err, "could not get to_branch_id for updating branch propagation spec")
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE pfs.branch_provenance
			SET never = $3
			WHERE from_id = $1
			AND to_id = $2
		`, branchID, toBranchID, bps.PropagationSpec.Never); err != nil {
			return branchID, errors.Wrap(err, "could not update branch propagation spec")
		}
	}
	return branchID, nil
}

// DeleteBranch deletes a branch.
func DeleteBranch(ctx context.Context, tx *pachsql.Tx, b *Branch, force bool) error {
	if !force {
		subv, err := GetDirectBranchSubvenance(ctx, tx, b.ID)
		if err != nil {
			return errors.Wrapf(err, "collect direct subvenance of branch %q", b.BranchInfo.Branch)
		}
		if len(subv) > 0 {
			return errors.Errorf(
				"branch %q cannot be deleted because it's in the direct provenance of %v",
				b.BranchInfo.Branch, subv,
			)
		}
		triggered, err := GetTriggeredBranches(ctx, tx, b.ID)
		if err != nil {
			return errors.Wrapf(err, "collect triggered branches for branch %q", b.BranchInfo.Branch)
		}
		if len(triggered) > 0 {
			return errors.Errorf(
				"branch %q cannot be deleted because it is triggered by branches %v",
				b.BranchInfo.Branch, triggered,
			)
		}
		triggering, err := GetTriggeringBranches(ctx, tx, b.ID)
		if err != nil {
			return errors.Wrapf(err, "collect triggering branches for branch %q", b.BranchInfo.Branch)
		}
		if len(triggering) > 0 {
			return errors.Errorf(
				"branch %q cannot be deleted because it triggers branches %v",
				b.BranchInfo.Branch, triggering,
			)
		}
	}
	deleteProvQuery := `DELETE FROM pfs.branch_provenance WHERE from_id = $1`
	deleteTriggerQuery := `DELETE FROM pfs.branch_triggers WHERE from_branch_id = $1`
	if force {
		deleteProvQuery = `DELETE FROM pfs.branch_provenance WHERE from_id = $1 OR to_id = $1`
		deleteTriggerQuery = `DELETE FROM pfs.branch_triggers WHERE from_branch_id = $1 OR to_branch_id = $1`
	}
	if _, err := tx.ExecContext(ctx, deleteProvQuery, b.ID); err != nil {
		return errors.Wrapf(err, "could not delete branch provenance for branch %d", b.BranchInfo.Branch)
	}
	if _, err := tx.ExecContext(ctx, deleteTriggerQuery, b.ID); err != nil {
		return errors.Wrapf(err, "could not delete branch trigger for branch %d", b.BranchInfo.Branch)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pfs.branches WHERE id = $1`, b.ID); err != nil {
		return errors.Wrapf(err, "could not delete branch %d", b.BranchInfo.Branch)
	}
	return nil
}

// GetDirectBranchProvenance returns the direct provenance of a branch, i.e. all branches that it directly depends on.
func GetDirectBranchProvenance(ctx context.Context, ext sqlx.ExtContext, id BranchID) ([]*pfs.Branch, error) {
	var branches []BranchRow
	if err := sqlx.SelectContext(ctx, ext, &branches, `
		SELECT
			branch.id,
			branch.name,
			repo.name as "repo.name",
			repo.type as "repo.type",
			project.name as "repo.project.name"
		FROM pfs.branch_provenance bp
		    JOIN pfs.branches branch ON bp.to_id = branch.id
			JOIN pfs.repos repo ON branch.repo_id = repo.id
			JOIN core.projects project ON repo.project_id = project.id
		WHERE bp.from_id = $1
	`, id); err != nil {
		return nil, errors.Wrap(err, "could not get direct branch provenance")
	}
	var branchPbs []*pfs.Branch
	for _, branch := range branches {
		branchPbs = append(branchPbs, branch.Pb())
	}
	return branchPbs, nil
}

// GetFullBranchProvenance returns the full provenance of a branch, i.e. all branches that it either directly or transitively depends on.
func GetFullBranchProvenance(ctx context.Context, ext sqlx.ExtContext, id BranchID, opts ...GraphOption) ([]*pfs.Branch, error) {
	branches, err := getProvenantBranchRows(ctx, ext, id, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "get branch provenance")
	}
	var branchPbs []*pfs.Branch
	for _, branch := range branches {
		branchPbs = append(branchPbs, branch.Pb())
	}
	return branchPbs, nil
}

// GetProvenantBranches is like GetFullBranchProvenance but returns a slice of Branch structs instead.
func GetProvenantBranches(ctx context.Context, ext sqlx.ExtContext, id BranchID, opts ...GraphOption) ([]*Branch, error) {
	provenantBranches, err := getProvenantBranchRows(ctx, ext, id, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "get branch with id provenance")
	}
	var branches []*Branch
	for _, branch := range provenantBranches {
		branchInfo, err := fetchBranchInfoByBranch(ctx, ext, branch)
		if err != nil {
			return nil, errors.Wrap(err, "get branch with ID provenance")
		}
		branches = append(branches, &Branch{
			ID:         branch.ID,
			BranchInfo: branchInfo,
		})
	}
	return branches, nil
}

func getProvenantBranchRows(ctx context.Context, ext sqlx.ExtContext, id BranchID, opts ...GraphOption) ([]*BranchRow, error) {
	graphOpts := defaultGraphOptions()
	for _, opt := range opts {
		opt(graphOpts)
	}
	var branches []*BranchRow
	if err := sqlx.SelectContext(ctx, ext, &branches, `
		WITH RECURSIVE prov(from_id, to_id) AS (
		    SELECT from_id, to_id, 1 as depth
		    FROM pfs.branch_provenance
		    WHERE from_id = $1
		UNION ALL
		    SELECT DISTINCT bp.from_id, bp.to_id, depth+1
		    FROM prov JOIN pfs.branch_provenance bp ON prov.to_id = bp.from_id
		    WHERE depth < $2
		)
		SELECT
		    branch.id,
			branch.name,
			repo.name as "repo.name",
			repo.type as "repo.type",
			project.name as "repo.project.name"
		FROM pfs.branches branch
		JOIN prov p ON branch.id = p.to_id
		JOIN pfs.repos repo ON branch.repo_id = repo.id
		JOIN core.projects project ON repo.project_id = project.id
		GROUP BY branch.id, branch.name, repo.name, repo.type, project.name
		ORDER BY MIN(depth) ASC LIMIT $3;`,
		id, graphOpts.maxDepth, graphOpts.limit); err != nil {
		return nil, errors.Wrap(err, "could not get branch provenance")
	}
	return branches, nil
}

func GetDirectBranchSubvenance(ctx context.Context, ext sqlx.ExtContext, id BranchID) ([]*pfs.Branch, error) {
	var branches []BranchRow
	if err := sqlx.SelectContext(ctx, ext, &branches, `
		SELECT
			branch.id,
			branch.name,
			repo.name as "repo.name",
			repo.type as "repo.type",
			project.name as "repo.project.name"
		FROM pfs.branch_provenance bp
		    JOIN pfs.branches branch ON bp.from_id = branch.id
			JOIN pfs.repos repo ON branch.repo_id = repo.id
			JOIN core.projects project ON repo.project_id = project.id
		WHERE bp.to_id = $1
	`, id); err != nil {
		return nil, errors.Wrap(err, "could not get direct branch subvenance")
	}
	var branchPbs []*pfs.Branch
	for _, branch := range branches {
		branchPbs = append(branchPbs, branch.Pb())
	}
	return branchPbs, nil
}

// GetFullBranchSubvenance returns the full subvenance of a branch, i.e. all branches that either directly or transitively depend on it.
func GetFullBranchSubvenance(ctx context.Context, ext sqlx.ExtContext, id BranchID, opts ...GraphOption) ([]*pfs.Branch, error) {
	branches, err := getSubvenantBranchRows(ctx, ext, id, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "get branch subvenance")
	}
	var branchPbs []*pfs.Branch
	for _, branch := range branches {
		branchPbs = append(branchPbs, branch.Pb())
	}
	return branchPbs, nil
}

// GetSubvenantBranches is like GetFullBranchSubvenance but returns a slice of Branch structs instead.
func GetSubvenantBranches(ctx context.Context, ext sqlx.ExtContext, id BranchID, opts ...GraphOption) ([]*Branch, error) {
	subvenantBranches, err := getSubvenantBranchRows(ctx, ext, id, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "get branch with ID subvenance")
	}
	var branches []*Branch
	for _, branch := range subvenantBranches {
		branchInfo, err := fetchBranchInfoByBranch(ctx, ext, branch)
		if err != nil {
			return nil, errors.Wrap(err, "get branch with ID subvenance")
		}
		branches = append(branches, &Branch{
			ID:         branch.ID,
			BranchInfo: branchInfo,
		})
	}
	return branches, nil
}

func getSubvenantBranchRows(ctx context.Context, ext sqlx.ExtContext, id BranchID, opts ...GraphOption) ([]*BranchRow, error) {
	graphOpts := defaultGraphOptions()
	for _, opt := range opts {
		opt(graphOpts)
	}
	var branches []*BranchRow
	if err := sqlx.SelectContext(ctx, ext, &branches, `
		WITH RECURSIVE subv(from_id, to_id) AS (
		    SELECT from_id, to_id, 1 as depth
		    FROM pfs.branch_provenance
		    WHERE to_id = $1
		UNION ALL
		    SELECT DISTINCT bp.from_id, bp.to_id, depth+1
		    FROM subv JOIN pfs.branch_provenance bp ON subv.from_id = bp.to_id
		    WHERE depth < $2
		)
		SELECT
		    branch.id,
			branch.name,
			repo.name as "repo.name",
			repo.type as "repo.type",
			project.name as "repo.project.name"
		FROM pfs.branches branch
		JOIN subv s ON branch.id = s.from_id
		JOIN pfs.repos repo ON branch.repo_id = repo.id
		JOIN core.projects project ON repo.project_id = project.id
		GROUP BY branch.id, branch.name, repo.name, repo.type, project.name
		ORDER BY MIN(depth) ASC LIMIT $3;`,
		id, graphOpts.maxDepth, graphOpts.limit); err != nil {
		return nil, errors.Wrap(err, "could not get branch subvenance")
	}
	return branches, nil
}

// CreateDirectBranchProvenance creates a provenance relationship between two branches.
func CreateDirectBranchProvenance(ctx context.Context, ext sqlx.ExtContext, from, to BranchID) error {
	if _, err := ext.ExecContext(ctx, `
		INSERT INTO pfs.branch_provenance(from_id, to_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, from, to); err != nil {
		return errors.Wrap(err, "could not add branch provenance")
	}
	return nil
}

// GetTriggeredBranches lists all the branches that are directly triggered by a branch
func GetTriggeredBranches(ctx context.Context, ext sqlx.ExtContext, bid BranchID) ([]*pfs.Branch, error) {
	var branches []BranchRow
	q := `SELECT branch.id,
			branch.name,
			repo.name as "repo.name",
			repo.type as "repo.type",
			project.name as "repo.project.name"
              FROM pfs.branches branch
                  JOIN pfs.repos repo ON branch.repo_id = repo.id
                  JOIN core.projects project ON repo.project_id = project.id
                  JOIN pfs.branch_triggers trigger ON trigger.to_branch_id = $1
              WHERE branch.id = trigger.from_branch_id
        `
	if err := sqlx.SelectContext(ctx, ext, &branches, q, bid); err != nil {
		return nil, errors.Wrap(err, "could not get triggered branches")
	}
	var branchPbs []*pfs.Branch
	for _, branch := range branches {
		branchPbs = append(branchPbs, branch.Pb())
	}
	return branchPbs, nil
}

// GetTriggeringBranches lists all the branches that would directly trigger a branch
func GetTriggeringBranches(ctx context.Context, ext sqlx.ExtContext, bid BranchID) ([]*pfs.Branch, error) {
	var branches []BranchRow
	q := `SELECT branch.id,
			branch.name,
			repo.name as "repo.name",
			repo.type as "repo.type",
			project.name as "repo.project.name"
              FROM pfs.branches branch
                  JOIN pfs.repos repo ON branch.repo_id = repo.id
                  JOIN core.projects project ON repo.project_id = project.id
                  JOIN pfs.branch_triggers trigger ON trigger.from_branch_id = $1
              WHERE branch.id = trigger.to_branch_id
        `
	if err := sqlx.SelectContext(ctx, ext, &branches, q, bid); err != nil {
		return nil, errors.Wrap(err, "could not get triggering branches")
	}
	var branchPbs []*pfs.Branch
	for _, branch := range branches {
		branchPbs = append(branchPbs, branch.Pb())
	}
	return branchPbs, nil
}

func GetBranchTrigger(ctx context.Context, ext sqlx.ExtContext, from BranchID) (*pfs.Trigger, error) {
	// TODO: should this handle more than one trigger?
	trigger := BranchTrigger{}
	if err := sqlx.GetContext(ctx, ext, &trigger, `
		SELECT
			branch.name as "to_branch.name",
			cron_spec,
			rate_limit_spec,
			size,
			num_commits,
			all_conditions
		FROM pfs.branch_triggers trigger
			JOIN pfs.branches branch ON trigger.to_branch_id = branch.id
		WHERE trigger.from_branch_id = $1
	`, from); err != nil {
		if errors.As(err, sql.ErrNoRows) {
			// Branches don't need to have triggers
			return nil, nil
		}
		return nil, errors.Wrap(err, "could not get branch trigger")
	}
	return trigger.Pb(), nil
}

func UpsertBranchTrigger(ctx context.Context, tx *pachsql.Tx, from BranchID, to BranchID, trigger *pfs.Trigger) error {
	if trigger == nil {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pfs.branch_triggers(from_branch_id, to_branch_id, cron_spec, rate_limit_spec, size, num_commits, all_conditions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (from_branch_id) DO UPDATE SET
			to_branch_id = EXCLUDED.to_branch_id,
			cron_spec = EXCLUDED.cron_spec,
			rate_limit_spec = EXCLUDED.rate_limit_spec,
			size = EXCLUDED.size,
			num_commits = EXCLUDED.num_commits,
			all_conditions = EXCLUDED.all_conditions;
	`,
		from,
		to,
		trigger.CronSpec,
		trigger.RateLimitSpec,
		trigger.Size,
		trigger.Commits,
		trigger.All,
	); err != nil {
		return errors.Wrapf(err, "could not create trigger for branch %d", from)
	}
	return nil
}

func DeleteBranchTrigger(ctx context.Context, tx *pachsql.Tx, from BranchID) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM pfs.branch_triggers
		WHERE from_branch_id = $1
	`, from); err != nil {
		return errors.Wrapf(err, "could not delete branch trigger for branch %d", from)
	}
	return nil
}

func GetBranchProvenanceRowsDirectSubvenance(ctx context.Context, ext sqlx.ExtContext, id BranchID) ([]*BranchProvenanceRow, error) {
	var rows []*BranchProvenanceRow
	if err := sqlx.SelectContext(ctx, ext, &rows, `
		SELECT from_id, to_id, never
		FROM pfs.branch_provenance
		WHERE to_id = $1
	`, id); err != nil {
		return nil, errors.Wrap(err, "could not get branch provenance rows direct subvenance")
	}
	return rows, nil
}

func fetchBranchInfoByBranch(ctx context.Context, ext sqlx.ExtContext, branch *BranchRow) (*pfs.BranchInfo, error) {
	if branch == nil {
		return nil, errors.Errorf("branch cannot be nil")
	}
	branchInfo := &pfs.BranchInfo{
		Branch:    branch.Pb(),
		Head:      branch.Head.Pb(),
		Metadata:  branch.Metadata.Data,
		CreatedBy: branch.CreatedBy.String,
		CreatedAt: timestamppb.New(branch.CreatedAt),
		UpdatedAt: timestamppb.New(branch.UpdatedAt),
	}
	var err error
	branchInfo.DirectProvenance, err = GetDirectBranchProvenance(ctx, ext, branch.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get direct branch provenance")
	}
	branchInfo.Provenance, err = GetFullBranchProvenance(ctx, ext, branch.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get full branch provenance")
	}
	branchInfo.Subvenance, err = GetFullBranchSubvenance(ctx, ext, branch.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get full branch subvenance")
	}
	// trigger info
	branchInfo.Trigger, err = GetBranchTrigger(ctx, ext, branch.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get branch trigger")
	}
	branchInfo.BranchPropagationSpecs, err = GetBranchPropagationSpecs(ctx, ext, branch.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get branch propagation specs")
	}
	return branchInfo, nil
}

func GetBranchPropagationSpecs(ctx context.Context, ext sqlx.ExtContext, id BranchID) ([]*pfs.BranchPropagationSpec, error) {
	var rows []*BranchProvenanceRow
	if err := sqlx.SelectContext(ctx, ext, &rows, `
               SELECT to_id, never
               FROM pfs.branch_provenance
               WHERE from_id = $1
       `, id); err != nil {
		return nil, errors.Wrap(err, "could not get branch provenance rows")
	}
	var bps []*pfs.BranchPropagationSpec
	for _, row := range rows {
		// Skip default branch propagation specs.
		if !row.Never {
			continue
		}
		branch := &BranchRow{}
		id := row.ToID
		if err := sqlx.GetContext(ctx, ext, branch, getBranchByIDQuery, id); err != nil {
			return nil, errors.Wrap(err, "could not get branch")
		}
		bps = append(bps, &pfs.BranchPropagationSpec{
			Branch:          branch.Pb(),
			PropagationSpec: &pfs.PropagationSpec{Never: row.Never},
		})
	}
	return bps, nil
}

// Helper functions for watching branches.
type branchUpsertHandler func(id BranchID, branchInfo *pfs.BranchInfo) error
type branchDeleteHandler func(id BranchID) error

func WatchBranchesInRepo(ctx context.Context, db *pachsql.DB, listener collection.PostgresListener, repoID RepoID, onUpsert branchUpsertHandler, onDelete branchDeleteHandler) error {
	watcher, err := postgres.NewWatcher(db, listener, randutil.UniqueString(fmt.Sprintf("watch-branches-in-repo-%d", repoID)), BranchesInRepoChannel(repoID))
	if err != nil {
		return err
	}
	defer watcher.Close()
	// Optimized query for getting branches in a repo.
	query := getBranchBaseQuery + fmt.Sprintf("\nWHERE %s = ?\nORDER BY %s ASC", BranchColumnRepoID, BranchColumnID)
	query = db.Rebind(query)
	snapshot := &BranchIterator{paginator: newPageIterator[BranchRow](ctx, query, []any{repoID}, 0, branchesPageSize, 0), ext: db}
	return watchBranches(ctx, db, snapshot, watcher.Watch(), onUpsert, onDelete)
}

func watchBranches(ctx context.Context, db *pachsql.DB, snapshot stream.Iterator[Branch], events <-chan *postgres.Event, onUpsert branchUpsertHandler, onDelete branchDeleteHandler) error {
	// Handle snapshot.
	if err := stream.ForEach[Branch](ctx, snapshot, func(b Branch) error {
		return onUpsert(b.ID, b.BranchInfo)
	}); err != nil {
		return err
	}
	// Handle events.
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return errors.New("watch branches: events channel closed")
			}
			if event.Err != nil {
				return event.Err
			}
			id := BranchID(event.Id)
			switch event.Type {
			case postgres.EventDelete:
				if err := onDelete(id); err != nil {
					return err
				}
			case postgres.EventInsert, postgres.EventUpdate:
				var branchInfo *pfs.BranchInfo
				if err := dbutil.WithTx(ctx, db, func(ctx context.Context, tx *pachsql.Tx) error {
					var err error
					branchInfo, err = GetBranchInfo(ctx, tx, id)
					if err != nil {
						return err
					}
					return nil
				}); err != nil {
					return err
				}
				if err := onUpsert(id, branchInfo); err != nil {
					return err
				}
			default:
				return errors.Errorf("unknown event type: %v", event.Type)
			}
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "watch branches")
		}
	}
}

func PickBranch(ctx context.Context, branchPicker *pfs.BranchPicker, tx *pachsql.Tx) (*Branch, error) {
	if branchPicker == nil || branchPicker.Picker == nil {
		return nil, errors.New("branch picker cannot be nil")
	}
	switch branchPicker.Picker.(type) {
	case *pfs.BranchPicker_Name:
		picker := branchPicker.GetName()
		repo, err := PickRepo(ctx, picker.Repo, tx)
		if err != nil {
			return nil, errors.Wrap(err, "picking branch")
		}
		branch, err := GetBranch(ctx, tx, &pfs.Branch{
			Repo: repo.RepoInfo.Repo,
			Name: picker.Name,
		})
		if err != nil {
			return nil, errors.Wrap(err, "picking branch")
		}
		return branch, nil
	default:
		return nil, errors.Errorf("branch picker is of an unknown type: %T", branchPicker.Picker)
	}
}
