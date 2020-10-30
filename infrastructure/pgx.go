package infrastructure

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/go-querystring/query"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/crypto-com/chainindex/appinterface"
	"github.com/crypto-com/chainindex/usecase"
)

var PostgresStmtBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type PgxConnLike interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

type PgxConnPoolConfig struct {
	PgConnConfig           `url:"-"`
	MaybeMaxConns          *int32         `url:"pool_max_conns,omitempty"`
	MaybeMinConns          *int32         `url:"pool_min_conns,omitempty"`
	MaybeMaxConnLifeTime   *time.Duration `url:"pool_max_conn_lifetime,omitempty"`
	MaybeMaxConnIdleTime   *time.Duration `url:"pool_max_conn_idle_time,omitempty"`
	MaybeHealthCheckPeriod *time.Duration `url:"pool_health_check_period,omitempty"`
}

func (config *PgxConnPoolConfig) ToURL() string {
	var authStr string
	if config.MaybeUsername != nil || config.MaybePassword != nil {
		authStr = *config.MaybeUsername + ":" + *config.MaybePassword + "@"
	}
	connStr := fmt.Sprintf("postgres://%s%s:%d/%s", authStr, config.Host, config.Port, config.Database)

	queryValues, err := query.Values(config)
	if err != nil {
		panic(fmt.Sprintf("error parsing Pgx connection config: %v", err))
	}
	if !config.SSL {
		queryValues.Set("sslmode", "disable")
	}

	queryStr := queryValues.Encode()
	if len(queryStr) == 0 {
		return connStr
	}
	return connStr + "?" + queryStr
}

type PgxConn struct {
	// pgxConn could be simple connection or connetion pool
	pgxConn PgxConnLike
}

func NewPgxConn(config PgConnConfig, logger usecase.Logger) (*PgxConn, error) {
	pgxConfig, err := pgx.ParseConfig(config.ToURL())
	if err != nil {
		return nil, err
	}
	pgxConfig.Logger = NewPgxLoggerAdapter(logger)

	conn, err := pgx.ConnectConfig(context.Background(), pgxConfig)
	if err != nil {
		return nil, err
	}

	return &PgxConn{
		conn,
	}, nil
}

func NewPgxConnPool(config PgxConnPoolConfig, logger usecase.Logger) (*PgxConn, error) {
	pgxConfig, err := pgxpool.ParseConfig(config.ToURL())
	if err != nil {
		return nil, err
	}
	pgxConfig.ConnConfig.Logger = NewPgxLoggerAdapter(logger)

	conn, err := pgxpool.ConnectConfig(context.Background(), pgxConfig)
	if err != nil {
		return nil, err
	}

	return &PgxConn{
		conn,
	}, nil
}

func (conn *PgxConn) Begin() (appinterface.RDbTx, error) {
	tx, err := conn.pgxConn.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	return &PgxRDbTx{
		tx,
	}, nil
}
func (conn *PgxConn) Exec(sql string, args ...interface{}) (appinterface.RDbExecResult, error) {
	result, err := conn.pgxConn.Exec(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRDbExecResult{
		result,
	}, nil
}
func (conn *PgxConn) Query(sql string, args ...interface{}) (appinterface.RDbRowsResult, error) {
	rows, err := conn.pgxConn.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRDbRowsResult{
		rows,
	}, nil
}
func (conn *PgxConn) QueryRow(sql string, args ...interface{}) appinterface.RDbRowResult {
	return &PgxRDbRowResult{
		row: conn.pgxConn.QueryRow(context.Background(), sql, args...),
	}
}

type PgxRDbTx struct {
	tx pgx.Tx
}

func (tx *PgxRDbTx) Exec(sql string, args ...interface{}) (appinterface.RDbExecResult, error) {
	commandTag, err := tx.tx.Exec(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRDbExecResult{
		commandTag,
	}, nil
}

func (tx *PgxRDbTx) Query(sql string, args ...interface{}) (appinterface.RDbRowsResult, error) {
	rows, err := tx.tx.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRDbRowsResult{
		rows,
	}, nil
}
func (tx *PgxRDbTx) QueryRow(sql string, args ...interface{}) appinterface.RDbRowResult {
	return &PgxRDbRowResult{
		row: tx.tx.QueryRow(context.Background(), sql, args...),
	}
}
func (tx *PgxRDbTx) Commit() error {
	return tx.tx.Commit(context.Background())
}
func (tx *PgxRDbTx) Rollback() error {
	return tx.tx.Rollback(context.Background())
}

type PgxRDbExecResult struct {
	commandTag pgconn.CommandTag
}

func (result *PgxRDbExecResult) RowsAffected() int64 {
	return result.commandTag.RowsAffected()
}
func (result *PgxRDbExecResult) IsInsert() bool {
	return result.commandTag.Insert()
}
func (result *PgxRDbExecResult) IsUpdate() bool {
	return result.commandTag.Update()
}
func (result *PgxRDbExecResult) IsDelete() bool {
	return result.commandTag.Delete()
}
func (result *PgxRDbExecResult) IsSelect() bool {
	return result.commandTag.Select()
}
func (result *PgxRDbExecResult) String() string {
	return result.commandTag.String()
}

type PgxRDbRowsResult struct {
	rows pgx.Rows
}

func (result *PgxRDbRowsResult) Close() {
	result.rows.Close()
}
func (result *PgxRDbRowsResult) Err() error {
	return result.rows.Err()
}
func (result *PgxRDbRowsResult) ExecResult() appinterface.RDbExecResult {
	return &PgxRDbExecResult{
		commandTag: result.rows.CommandTag(),
	}
}
func (result *PgxRDbRowsResult) Next() bool {
	return result.rows.Next()
}
func (result *PgxRDbRowsResult) Scan(dest ...interface{}) error {
	err := result.rows.Scan(dest...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return appinterface.ErrNoRows
		}
		return err
	}
	return nil
}

type PgxRDbRowResult struct {
	row pgx.Row
}

func (result *PgxRDbRowResult) Scan(dest ...interface{}) error {
	err := result.row.Scan(dest...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return appinterface.ErrNoRows
		}
		return err
	}
	return nil
}

type PgxLoggerAdapter struct {
	logger usecase.Logger
}

func NewPgxLoggerAdapter(logger usecase.Logger) *PgxLoggerAdapter {
	return &PgxLoggerAdapter{
		logger: logger.WithFields(usecase.LogFields{
			"module": "pgx",
		}),
	}
}

func (logger *PgxLoggerAdapter) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	contextedLogger := logger.logger.WithFields(data)

	switch level {
	case pgx.LogLevelError:
		contextedLogger.Error(msg)
	case pgx.LogLevelWarn:
		contextedLogger.Info(msg)
	case pgx.LogLevelInfo:
		fallthrough
	case pgx.LogLevelDebug:
		fallthrough
	case pgx.LogLevelNone:
		fallthrough
	default:
		contextedLogger.Debug(msg)
	}
}