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

func (crossfireValidatorsStatsView *CrossfireValidatorsStats) Set(key string, value int64) error {
	// UPSERT STATEMENT
	sql, sqlArgs, err := crossfireValidatorsStatsView.rdbHandle.StmtBuilder.
		Insert(CROSSFIRE_VALIDATOR_STATS_VIEW_TABLENAME).
		Columns("key", "value").
		Values(key, value).
		Suffix("ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value").
		ToSql()
	if err != nil {
		return fmt.Errorf("error building value insertion sql: %v: %w", err, rdb.ErrBuildSQLStmt)
	}

	_, err = crossfireValidatorsStatsView.rdbHandle.Exec(sql, sqlArgs...)

	if err != nil {
		return fmt.Errorf("error inserting value: %v: %w", err, rdb.ErrWrite)
	}

	return nil
}

func (crossfireValidatorsStatsView *CrossfireValidatorsStats) Increment(key string, value int64) error {
	// Postgres UPSERT statement

	getMetric, err := crossfireValidatorsStatsView.FindBy(key)
	if err != nil {
		return fmt.Errorf("error getting value: %v: %w", err, rdb.ErrBuildSQLStmt)
	}
	if getMetric <= 0 {
		return fmt.Errorf("Got empty value! : %v", getMetric)
	}
	sql, sqlArgs, err := crossfireValidatorsStatsView.rdbHandle.StmtBuilder.
		Insert(CROSSFIRE_VALIDATOR_STATS_VIEW_TABLENAME+" AS totals").
		Columns("key", "value").
		Values(key, value).
		Suffix("ON CONFLICT (key) DO UPDATE SET value = totals.value + EXCLUDED.value").
		ToSql()
	if err != nil {
		return fmt.Errorf("error building value insertion sql: %v: %w", err, rdb.ErrBuildSQLStmt)
	}

	_, err = crossfireValidatorsStatsView.rdbHandle.Exec(sql, sqlArgs...)

	if err != nil {
		return fmt.Errorf("error inserting value: %v: %w", err, rdb.ErrWrite)
	}

	return nil
}

func (crossfireValidatorsStatsView *CrossfireValidatorsStats) FindBy(key string) (int64, error) {
	sql, sqlArgs, err := crossfireValidatorsStatsView.rdbHandle.StmtBuilder.Select(
		"value",
	).From(
		CROSSFIRE_VALIDATOR_STATS_VIEW_TABLENAME,
	).Where(
		"key = ?", key,
	).ToSql()
	if err != nil {
		return -1, fmt.Errorf("error preparing key selection SQL: %v", err)
	}

	var value int64
	if err := crossfireValidatorsStatsView.rdbHandle.QueryRow(sql, sqlArgs...).Scan(&value); err != nil {
		if errors.Is(err, rdb.ErrNoRows) {
			return -1, nil
		}
		return -1, fmt.Errorf("error getting key: %v", err)
	}

	return value, nil
}

type CrossfireValidatorsStatsRow struct {
	Key   string
	Value int64
}