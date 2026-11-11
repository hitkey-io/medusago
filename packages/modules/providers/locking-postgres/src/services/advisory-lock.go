// Package services mirrors @medusajs/locking-postgres/src/services.
package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/providers"
)

// Identifier mirrors PostgresAdvisoryLockProvider.identifier.
const Identifier = "locking-postgres"

// PostgresAdvisoryLockProvider mirrors PostgresAdvisoryLockProvider:
// execute holds transaction-scoped advisory locks (keys hashed like
// upstream, so Node and Go instances exclude each other); acquire/release
// use rows of the "locking" table.
type PostgresAdvisoryLockProvider struct {
	db *pgxpool.Pool
}

var _ lockingtypes.ILockingProvider = (*PostgresAdvisoryLockProvider)(nil)

// NewPostgresAdvisoryLockProvider builds the provider over the pool.
func NewPostgresAdvisoryLockProvider(db *pgxpool.Pool) *PostgresAdvisoryLockProvider {
	return &PostgresAdvisoryLockProvider{db: db}
}

// HashStringToInt mirrors hashStringToInt: djb2 (xor variant) over the
// UTF-16 code units from the end, as an unsigned 32-bit integer.
func HashStringToInt(s string) uint32 {
	units := utf16.Encode([]rune(s))
	hash := int32(5381)
	for i := len(units) - 1; i >= 0; i-- {
		hash = int32(int64(hash)*33) ^ int32(units[i])
	}
	return uint32(hash)
}

// Execute mirrors execute: in one transaction, pg_advisory_xact_lock for
// every key (racing the timeout), then the job, then commit (rollback when
// the job fails).
func (p *PostgresAdvisoryLockProvider) Execute(ctx context.Context, keys []string, job lockingtypes.Job, args lockingtypes.ExecuteArgs) (any, error) {
	timeoutSeconds := providers.ExecuteTimeout(args.Timeout, 5)

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	lockCtx, cancel := context.WithTimeout(ctx, providers.SecondsDuration(timeoutSeconds))
	var lockErr error
	for _, key := range keys {
		if _, err := tx.Exec(lockCtx, "SELECT pg_advisory_xact_lock($1)", int64(HashStringToInt(key))); err != nil {
			lockErr = err
			break
		}
	}
	timedOut := lockCtx.Err() == context.DeadlineExceeded
	cancel()
	if lockErr != nil {
		_ = tx.Rollback(context.Background())
		if timedOut {
			return nil, errors.New("Timed-out acquiring lock.")
		}
		return nil, lockErr
	}

	ret, err := job(ctx)
	if err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return ret, nil
}

type lockRow struct {
	ownerID    *string
	expiration *time.Time
	now        time.Time
}

// loadLock mirrors loadLock (nil when there is no row).
func (p *PostgresAdvisoryLockProvider) loadLock(ctx context.Context, key string) (*lockRow, error) {
	var r lockRow
	err := p.db.QueryRow(ctx, `SELECT owner_id, expiration, NOW() AS now FROM locking WHERE id = $1`, key).Scan(&r.ownerID, &r.expiration, &r.now)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// looseEqual mirrors `a == b` between owner ids (null == undefined).
func looseEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// jsNumber formats a number like `${+n}`.
func jsNumber(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// Acquire mirrors acquire.
func (p *PostgresAdvisoryLockProvider) Acquire(ctx context.Context, keys []string, args lockingtypes.AcquireArgs) error {
	ownerID, expire := args.OwnerID, args.Expire
	for _, key := range keys {
		row, err := p.loadLock(ctx, key)
		if err != nil {
			return err
		}
		if row == nil {
			expireSQL := "NULL"
			if expire != 0 {
				expireSQL = "NOW() + INTERVAL '" + jsNumber(expire) + " SECONDS'"
			}
			_, err := p.db.Exec(ctx, `INSERT INTO locking (id, owner_id, expiration) VALUES ($1, $2, `+expireSQL+`)`, key, ownerID)
			if err != nil {
				if !strings.Contains(err.Error(), "locking_pkey") {
					return err
				}
				owner, lerr := p.loadLock(ctx, key)
				if lerr != nil {
					return lerr
				}
				if owner == nil {
					return errors.New("Cannot read properties of undefined (reading 'owner_id')")
				}
				if !looseEqual(ownerID, owner.ownerID) {
					return fmt.Errorf(`"%s" is already locked.`, key)
				}
			}
			continue
		}

		if row.ownerID == nil || ownerID == nil || *row.ownerID != *ownerID {
			return fmt.Errorf(`Failed to acquire lock for key "%s"`, key)
		}
		if row.expiration == nil {
			continue
		}
		// canRefresh = same owner && (expire || expired): only an expire
		// refreshes the row
		if expire == 0 {
			continue
		}
		if _, err := p.db.Exec(ctx, `UPDATE locking SET owner_id = $1, expiration = NOW() + INTERVAL '`+jsNumber(expire)+` SECONDS' WHERE id = $2`, ownerID, key); err != nil {
			return err
		}
	}
	return nil
}

// Release mirrors release: false when a row is missing, owned by someone
// else or already expired.
func (p *PostgresAdvisoryLockProvider) Release(ctx context.Context, keys []string, args lockingtypes.ReleaseArgs) (bool, error) {
	success := true
	for _, key := range keys {
		row, err := p.loadLock(ctx, key)
		if err != nil {
			return false, err
		}
		if row == nil || !looseEqual(row.ownerID, args.OwnerID) {
			success = false
			continue
		}
		if _, err := p.db.Exec(ctx, `DELETE FROM locking WHERE id = $1`, key); err != nil {
			return false, err
		}
		success = success && (row.expiration == nil || row.expiration.After(row.now))
	}
	return success, nil
}

// ReleaseAll mirrors releaseAll.
func (p *PostgresAdvisoryLockProvider) ReleaseAll(ctx context.Context, args lockingtypes.ReleaseArgs) error {
	if args.OwnerID == nil {
		_, err := p.db.Exec(ctx, `TRUNCATE TABLE locking`)
		return err
	}
	_, err := p.db.Exec(ctx, `DELETE FROM locking WHERE owner_id = $1`, *args.OwnerID)
	return err
}
