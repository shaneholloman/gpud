// Package state provides the persistent storage layer for the log poller.
package state

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/leptonai/gpud/pkg/sqlite"
)

const TableNameLogFileSeekInfo = "components_query_log_seek_info"

const (
	ColumnFile = "file"

	// File seek info offset.
	ColumnOffset = "offset"
	// File seek info whence.
	ColumnWhence = "whence"
)

func CreateTableLogFileSeekInfo(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	%s TEXT NOT NULL PRIMARY KEY,
	%s INTEGER NOT NULL,
	%s INTEGER NOT NULL
);`, TableNameLogFileSeekInfo, ColumnFile, ColumnOffset, ColumnWhence))
	return err
}

func InsertLogFileSeekInfo(ctx context.Context, db *sql.DB, file string, offset int64, whence int64) error {
	query := fmt.Sprintf(`
INSERT OR REPLACE INTO %s (%s, %s, %s) VALUES (?, ?, ?);
`,
		TableNameLogFileSeekInfo,
		ColumnFile,
		ColumnOffset,
		ColumnWhence,
	)

	start := time.Now()
	_, err := db.ExecContext(ctx, query, file, offset, whence)
	sqlite.RecordInsertUpdate(time.Since(start).Seconds())

	return err
}

// Returns "database/sql.ErrNoRows" if no record is found.
func GetLogFileSeekInfo(ctx context.Context, db *sql.DB, file string) (int64, int64, error) {
	query := fmt.Sprintf(`SELECT %s, %s FROM %s WHERE %s = ?;`, ColumnOffset, ColumnWhence, TableNameLogFileSeekInfo, ColumnFile)

	start := time.Now()
	row := db.QueryRowContext(ctx, query, file)
	sqlite.RecordSelect(time.Since(start).Seconds())

	var offset, whence int64
	err := row.Scan(&offset, &whence)
	return offset, whence, err
}
