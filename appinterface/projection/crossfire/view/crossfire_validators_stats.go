package view

import (
	"errors"
	"fmt"

	"github.com/crypto-com/chain-indexing/appinterface/rdb"
)

type CrossfireValidatorsStats struct {
	rdbHandle *rdb.Handle
}

const CROSSFIRE_VALIDATOR_STATS_VIEW_TABLENAME = "view_crossfire_validators_stats"

func NewCrossfireValidatorsStats(handle *rdb.Handle) *CrossfireValidatorsStats {
	return &CrossfireValidatorsStats{
		handle,
	}
}

func (view *CrossfireValidatorsStats) Set(key string, value int64) error {
	// UPSERT STATEMENT
	sql, sqlArgs, err := view.rdbHandle.StmtBuilder.
		Insert(CROSSFIRE_VALIDATOR_STATS_VIEW_TABLENAME).
		Columns("key", "value").
		Values(key, value).
		Suffix("ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value").
		ToSql()
	if err != nil {
		return fmt.Errorf("error building value insertion sql: %v: %w", err, rdb.ErrBuildSQLStmt)
	}

	_, err = view.rdbHandle.Exec(sql, sqlArgs...)

	if err != nil {
		return fmt.Errorf("error inserting value: %v: %w", err, rdb.ErrWrite)
	}

	return nil
}

func (view *CrossfireValidatorsStats) FindBy(key string) (int64, error) {
	sql, sqlArgs, err := view.rdbHandle.StmtBuilder.Select(
		"value",
	).From(
		CROSSFIRE_VALIDATOR_STATS_VIEW_TABLENAME,
	).Where(
		"key = ?", key,
	).ToSql()
	if err != nil {
		return 0, fmt.Errorf("error preparing key selection SQL: %v", err)
	}

	var value int64
	if err := view.rdbHandle.QueryRow(sql, sqlArgs...).Scan(&value); err != nil {
		if errors.Is(err, rdb.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("error getting key: %v", err)
	}

	return value, nil
}

func (view *CrossfireValidatorsStats) IncrementOne(key string) error {
	value, err := view.FindBy(key)
	if err != nil {
		return fmt.Errorf("error getting value for %v: %v", key, err)
	}

	err = view.Set(key, value + 1)
	if err != nil {
		return fmt.Errorf("error setting increment for %v: %v", key, err)
	}

	return nil
}

type CrossfireValidatorsStatsRow struct {
	Key   string
	Value int64
}
