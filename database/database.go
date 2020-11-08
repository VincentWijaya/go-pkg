package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Config struct {
	// data source name
	// eg:
	// postgresql: host=localhost port=5432 user=luckyong password=mysecretpassword dbname=playground sslmode=disable
	// mysql: droplet_write:Komodo2019@tcp(192.169.2.26:3306)/droplet
	DSN string

	// postgres, mysql, cockroachdb, etc
	Driver string

	// set maximum open connection in pool
	// by default there is no maximum number to open connection to db
	MaxOpenConns int

	// set maximum idle connections
	// by default connections that are not used are mark idle and then closed
	MaxIdleConns int

	// set maximum connection lifetime (in hour)
	// by default the connection will never expired
	ConnMaxLifeTime int
}

type Database struct {
	connection *sqlx.DB
}

type Statement struct {
	statement *sqlx.Stmt
}

type NamedStatement struct {
	statement *sqlx.NamedStmt
}

type DBTransaction struct {
	connection  *sqlx.DB
	transaction *sqlx.Tx
}

type DB interface {
	Ping() error
	Rebind(query string) string
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQueryRowx(ctx context.Context, query string, arg interface{}) *sqlx.Row
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedGet(ctx context.Context, dest interface{}, query string, arg interface{}) error
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedSelect(ctx context.Context, dest interface{}, query string, arg interface{}) error
	Begin() (Tx, error)
	Prepare(ctx context.Context, query string) (Stmt, error)
	NamedPrepare(ctx context.Context, query string) (Stmt, error)
}

type Stmt interface {
	Exec(ctx context.Context, args ...interface{}) (sql.Result, error)
	Get(ctx context.Context, dest interface{}, args ...interface{}) error
	Select(ctx context.Context, dest interface{}, args ...interface{}) error
}

type Row interface {
	Scan(args ...interface{}) error
}

type Tx interface {
	Commit() error
	Rollback() error
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQueryRowx(ctx context.Context, query string, arg interface{}) *sqlx.Row
}

// ErrNoRows postgresql error return no result set
var ErrNoRows = sql.ErrNoRows

// Connect open connection to
func Connect(cfg Config) (DB, error) {
	db, err := sqlx.Connect(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, err
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	if cfg.ConnMaxLifeTime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifeTime) * time.Hour)
	}

	return &Database{
		connection: db,
	}, db.Ping()
}

func convertNamed(query string, arg interface{}) (string, []interface{}, error) {
	query, args, err := sqlx.Named(query, arg)
	if err != nil {
		return query, args, err
	}

	return sqlx.In(query, args...)
}

func (db *Database) Ping() error {
	return db.connection.Ping()
}

// Rebind to get a query which is suitable bindvar syntax (query placeholder) for execution
func (db *Database) Rebind(query string) string {
	return db.connection.Rebind(query)
}

func (db *Database) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	query = db.connection.Rebind(query)
	return db.connection.ExecContext(ctx, query, args...)
}

func (db *Database) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	query, args, err := convertNamed(query, arg)
	if err != nil {
		return nil, err
	}
	query = db.connection.Rebind(query)
	return db.connection.ExecContext(ctx, query, args...)
}

func (db *Database) NamedQueryRowx(ctx context.Context, query string, arg interface{}) *sqlx.Row {
	query, args, err := convertNamed(query, arg)
	if err != nil {
		return nil
	}
	query = db.connection.Rebind(query)
	return db.connection.QueryRowxContext(ctx, query, args...)
}

func (db *Database) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return db.connection.GetContext(ctx, dest, query, args...)
}

func (db *Database) NamedGet(ctx context.Context, dest interface{}, query string, arg interface{}) error {
	query, args, err := convertNamed(query, arg)
	if err != nil {
		return err
	}
	query = db.connection.Rebind(query)
	return db.connection.GetContext(ctx, dest, query, args...)
}

func (db *Database) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return db.connection.SelectContext(ctx, dest, query, args...)
}

func (db *Database) NamedSelect(ctx context.Context, dest interface{}, query string, arg interface{}) error {
	query, args, err := convertNamed(query, arg)
	if err != nil {
		return err
	}
	query = db.connection.Rebind(query)
	return db.connection.SelectContext(ctx, dest, query, args...)
}

func (db *Database) Begin() (Tx, error) {
	tx, err := db.connection.Beginx()
	if err != nil {
		return nil, err
	}
	return &DBTransaction{transaction: tx, connection: db.connection}, nil
}

func (tx *DBTransaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return tx.transaction.ExecContext(ctx, query, args...)
}

func (tx *DBTransaction) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	query, args, err := convertNamed(query, arg)
	if err != nil {
		return nil, err
	}
	query = tx.connection.Rebind(query)
	return tx.transaction.ExecContext(ctx, query, args...)
}

func (tx *DBTransaction) NamedQueryRowx(ctx context.Context, query string, arg interface{}) *sqlx.Row {
	query, args, err := convertNamed(query, arg)
	if err != nil {
		return nil
	}
	query = tx.connection.Rebind(query)
	return tx.transaction.QueryRowxContext(ctx, query, args...)
}

func (tx *DBTransaction) Commit() error {
	return tx.transaction.Commit()
}

func (tx *DBTransaction) Rollback() error {
	return tx.transaction.Rollback()
}

func (db *Database) Prepare(ctx context.Context, query string) (Stmt, error) {
	stmt, err := db.connection.PreparexContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &Statement{statement: stmt}, nil
}

func (stmt *Statement) Exec(ctx context.Context, args ...interface{}) (sql.Result, error) {
	return stmt.statement.ExecContext(ctx, args...)
}

func (stmt *Statement) Get(ctx context.Context, dest interface{}, args ...interface{}) error {
	return stmt.statement.GetContext(ctx, dest, args...)
}

func (stmt *Statement) Select(ctx context.Context, dest interface{}, args ...interface{}) error {
	return stmt.statement.SelectContext(ctx, dest, args...)
}

func (db *Database) NamedPrepare(ctx context.Context, query string) (Stmt, error) {
	stmt, err := db.connection.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &NamedStatement{statement: stmt}, nil
}

func (stmt *NamedStatement) Exec(ctx context.Context, args ...interface{}) (sql.Result, error) {
	if len(args) == 0 {
		return nil, errors.New("Missing parameter for this action")
	}
	return stmt.statement.ExecContext(ctx, args[0])
}

func (stmt *NamedStatement) Get(ctx context.Context, dest interface{}, args ...interface{}) error {
	if len(args) == 0 {
		return errors.New("Missing parameter for this action")
	}
	return stmt.statement.GetContext(ctx, dest, args[0])
}

func (stmt *NamedStatement) Select(ctx context.Context, dest interface{}, args ...interface{}) error {
	if len(args) == 0 {
		return errors.New("Missing parameter for this action")
	}
	return stmt.statement.SelectContext(ctx, dest, args[0])
}
