package trdsql

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"time"
	"unicode/utf8"
)

// Exporter is the interface for processing query results.
// Exporter executes SQL and outputs to Writer.
type Exporter interface {
	Export(db *DB, query string) error
}

// WriteFormat represents a structure that satisfies Exporter.
type WriteFormat struct {
	Writer
}

// NewExporter returns trdsql default Exporter.
func NewExporter(writer Writer) *WriteFormat {
	return &WriteFormat{
		Writer: writer,
	}
}

// Export is execute SQL(Select) and
// the result is written out by the writer.
// Export is called from Exec.
func (e *WriteFormat) Export(db *DB, query string) error {
	rows, err := db.Select(query)
	if err != nil {
		return err
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			log.Printf("ERROR: close:%s", err)
		}
	}()
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	types := make([]string, len(columns))
	for i, ct := range columnTypes {
		types[i] = ct.DatabaseTypeName()
	}

	err = e.Writer.PreWrite(columns, types)
	if err != nil {
		return err
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return err
		}
		err = e.Writer.WriteRow(values, columns)
		if err != nil {
			return err
		}
	}
	return e.Writer.PostWrite()
}

// ValString converts database value to string.
func ValString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		if ok := utf8.Valid(t); ok {
			return string(t)
		}
		return `\x` + hex.EncodeToString(t)
	case int:
		return strconv.Itoa(t)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}
