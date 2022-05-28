package models

// Code generated by xo. DO NOT EDIT.

import (
	"context"
	"database/sql"
)

// Run represents a row from 'runs'.
type Run struct {
	ID          string        `json:"id"`           // id
	UserID      string        `json:"user_id"`      // user_id
	ThunkDigest string        `json:"thunk_digest"` // thunk_digest
	StartTime   Time          `json:"start_time"`   // start_time
	EndTime     *Time         `json:"end_time"`     // end_time
	Succeeded   sql.NullInt64 `json:"succeeded"`    // succeeded
	// xo fields
	_exists, _deleted bool
}

// Exists returns true when the Run exists in the database.
func (r *Run) Exists() bool {
	return r._exists
}

// Deleted returns true when the Run has been marked for deletion from
// the database.
func (r *Run) Deleted() bool {
	return r._deleted
}

// Insert inserts the Run to the database.
func (r *Run) Insert(ctx context.Context, db DB) error {
	switch {
	case r._exists: // already exists
		return logerror(&ErrInsertFailed{ErrAlreadyExists})
	case r._deleted: // deleted
		return logerror(&ErrInsertFailed{ErrMarkedForDeletion})
	}
	// insert (manual)
	const sqlstr = `INSERT INTO runs (` +
		`id, user_id, thunk_digest, start_time, end_time, succeeded` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6` +
		`)`
	// run
	logf(sqlstr, r.ID, r.UserID, r.ThunkDigest, r.StartTime, r.EndTime, r.Succeeded)
	if _, err := db.ExecContext(ctx, sqlstr, r.ID, r.UserID, r.ThunkDigest, r.StartTime, r.EndTime, r.Succeeded); err != nil {
		return logerror(err)
	}
	// set exists
	r._exists = true
	return nil
}

// Update updates a Run in the database.
func (r *Run) Update(ctx context.Context, db DB) error {
	switch {
	case !r._exists: // doesn't exist
		return logerror(&ErrUpdateFailed{ErrDoesNotExist})
	case r._deleted: // deleted
		return logerror(&ErrUpdateFailed{ErrMarkedForDeletion})
	}
	// update with primary key
	const sqlstr = `UPDATE runs SET ` +
		`user_id = $1, thunk_digest = $2, start_time = $3, end_time = $4, succeeded = $5 ` +
		`WHERE id = $6`
	// run
	logf(sqlstr, r.UserID, r.ThunkDigest, r.StartTime, r.EndTime, r.Succeeded, r.ID)
	if _, err := db.ExecContext(ctx, sqlstr, r.UserID, r.ThunkDigest, r.StartTime, r.EndTime, r.Succeeded, r.ID); err != nil {
		return logerror(err)
	}
	return nil
}

// Save saves the Run to the database.
func (r *Run) Save(ctx context.Context, db DB) error {
	if r.Exists() {
		return r.Update(ctx, db)
	}
	return r.Insert(ctx, db)
}

// Upsert performs an upsert for Run.
func (r *Run) Upsert(ctx context.Context, db DB) error {
	switch {
	case r._deleted: // deleted
		return logerror(&ErrUpsertFailed{ErrMarkedForDeletion})
	}
	// upsert
	const sqlstr = `INSERT INTO runs (` +
		`id, user_id, thunk_digest, start_time, end_time, succeeded` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6` +
		`)` +
		` ON CONFLICT (id) DO ` +
		`UPDATE SET ` +
		`user_id = EXCLUDED.user_id, thunk_digest = EXCLUDED.thunk_digest, start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, succeeded = EXCLUDED.succeeded `
	// run
	logf(sqlstr, r.ID, r.UserID, r.ThunkDigest, r.StartTime, r.EndTime, r.Succeeded)
	if _, err := db.ExecContext(ctx, sqlstr, r.ID, r.UserID, r.ThunkDigest, r.StartTime, r.EndTime, r.Succeeded); err != nil {
		return logerror(err)
	}
	// set exists
	r._exists = true
	return nil
}

// Delete deletes the Run from the database.
func (r *Run) Delete(ctx context.Context, db DB) error {
	switch {
	case !r._exists: // doesn't exist
		return nil
	case r._deleted: // deleted
		return nil
	}
	// delete with single primary key
	const sqlstr = `DELETE FROM runs ` +
		`WHERE id = $1`
	// run
	logf(sqlstr, r.ID)
	if _, err := db.ExecContext(ctx, sqlstr, r.ID); err != nil {
		return logerror(err)
	}
	// set deleted
	r._deleted = true
	return nil
}

// RunsByThunkDigest retrieves a row from 'runs' as a Run.
//
// Generated from index 'idx_thunk_runs_digest'.
func RunsByThunkDigest(ctx context.Context, db DB, thunkDigest string) ([]*Run, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, user_id, thunk_digest, start_time, end_time, succeeded ` +
		`FROM runs ` +
		`WHERE thunk_digest = $1`
	// run
	logf(sqlstr, thunkDigest)
	rows, err := db.QueryContext(ctx, sqlstr, thunkDigest)
	if err != nil {
		return nil, logerror(err)
	}
	defer rows.Close()
	// process
	var res []*Run
	for rows.Next() {
		r := Run{
			_exists: true,
		}
		// scan
		if err := rows.Scan(&r.ID, &r.UserID, &r.ThunkDigest, &r.StartTime, &r.EndTime, &r.Succeeded); err != nil {
			return nil, logerror(err)
		}
		res = append(res, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, logerror(err)
	}
	return res, nil
}

// RunsByUserID retrieves a row from 'runs' as a Run.
//
// Generated from index 'idx_thunk_runs_user_id'.
func RunsByUserID(ctx context.Context, db DB, userID string) ([]*Run, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, user_id, thunk_digest, start_time, end_time, succeeded ` +
		`FROM runs ` +
		`WHERE user_id = $1`
	// run
	logf(sqlstr, userID)
	rows, err := db.QueryContext(ctx, sqlstr, userID)
	if err != nil {
		return nil, logerror(err)
	}
	defer rows.Close()
	// process
	var res []*Run
	for rows.Next() {
		r := Run{
			_exists: true,
		}
		// scan
		if err := rows.Scan(&r.ID, &r.UserID, &r.ThunkDigest, &r.StartTime, &r.EndTime, &r.Succeeded); err != nil {
			return nil, logerror(err)
		}
		res = append(res, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, logerror(err)
	}
	return res, nil
}

// RunByID retrieves a row from 'runs' as a Run.
//
// Generated from index 'sqlite_autoindex_runs_1'.
func RunByID(ctx context.Context, db DB, id string) (*Run, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, user_id, thunk_digest, start_time, end_time, succeeded ` +
		`FROM runs ` +
		`WHERE id = $1`
	// run
	logf(sqlstr, id)
	r := Run{
		_exists: true,
	}
	if err := db.QueryRowContext(ctx, sqlstr, id).Scan(&r.ID, &r.UserID, &r.ThunkDigest, &r.StartTime, &r.EndTime, &r.Succeeded); err != nil {
		return nil, logerror(err)
	}
	return &r, nil
}

// Thunk returns the Thunk associated with the Run's (ThunkDigest).
//
// Generated from foreign key 'runs_thunk_digest_fkey'.
func (r *Run) Thunk(ctx context.Context, db DB) (*Thunk, error) {
	return ThunkByDigest(ctx, db, r.ThunkDigest)
}

// User returns the User associated with the Run's (UserID).
//
// Generated from foreign key 'runs_user_id_fkey'.
func (r *Run) User(ctx context.Context, db DB) (*User, error) {
	return UserByID(ctx, db, r.UserID)
}
