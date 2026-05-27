/*-------------------------------------------------------------------------
 *
 * pgEdge Natural Language Agent
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

// Helpers for loading database schema metadata. The query lives in
// load_metadata.sql so it can be diffed and reviewed independently; the
// per-row scan and the (rows -> TableInfo map) transformation are split
// into their own functions so the latter is testable without a database.

package database

import (
	"database/sql"
	_ "embed"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"
)

//go:embed load_metadata.sql
var loadMetadataSQL string

// vectorDimsRe extracts the dimension count from a pgvector type
// descriptor like "vector(1536)".
var vectorDimsRe = regexp.MustCompile(`vector\((\d+)\)`)

// metadataRow is one scanned row of loadMetadataSQL. Columns originating
// from the LEFT JOIN against column_info are nullable because a table
// with zero columns still produces a row with all column_info fields
// NULL (see issue #126).
type metadataRow struct {
	SchemaName    string
	TableName     string
	TableType     string
	TableDesc     string
	IsPartitioned bool
	IsPartition   bool
	ColumnName    sql.NullString
	DataType      sql.NullString
	IsNullable    sql.NullString
	ColumnDesc    string
	TypeName      sql.NullString
	TypeModifier  sql.NullInt32
	IsPrimaryKey  bool
	IsUnique      bool
	FKReference   string
	IsIndexed     bool
	IdentityType  string
	DefaultValue  string
}

// scanMetadataRow scans one row of loadMetadataSQL into a metadataRow.
// The order of the Scan arguments must match the SELECT list of the
// query exactly.
func scanMetadataRow(rows pgx.Rows) (metadataRow, error) {
	var r metadataRow
	err := rows.Scan(
		&r.SchemaName,
		&r.TableName,
		&r.TableType,
		&r.TableDesc,
		&r.IsPartitioned,
		&r.IsPartition,
		&r.ColumnName,
		&r.DataType,
		&r.IsNullable,
		&r.ColumnDesc,
		&r.TypeName,
		&r.TypeModifier,
		&r.IsPrimaryKey,
		&r.IsUnique,
		&r.FKReference,
		&r.IsIndexed,
		&r.IdentityType,
		&r.DefaultValue,
	)
	return r, err
}

// buildTableInfo groups scanned rows by (schema, table) and constructs
// the metadata map. The returned map is keyed "schema.table". schemas
// collects the distinct schema names seen. columnCount is the total
// number of column entries appended across all tables — rows with a
// NULL or empty column name (tables with no columns) contribute a
// TableInfo with Columns == [] but do not increment columnCount.
//
// This function is pure: no I/O, no time, no shared state.
func buildTableInfo(rows []metadataRow) (metadata map[string]TableInfo, schemas map[string]bool, columnCount int) {
	metadata = make(map[string]TableInfo)
	schemas = make(map[string]bool)

	for _, r := range rows {
		key := r.SchemaName + "." + r.TableName
		schemas[r.SchemaName] = true

		table, exists := metadata[key]
		if !exists {
			table = TableInfo{
				SchemaName:    r.SchemaName,
				TableName:     r.TableName,
				TableType:     r.TableType,
				Description:   r.TableDesc,
				IsPartitioned: r.IsPartitioned,
				IsPartition:   r.IsPartition,
				Columns:       []ColumnInfo{},
			}
		}

		if r.ColumnName.Valid && r.ColumnName.String != "" {
			isVector := false
			dimensions := 0
			if r.TypeName.Valid && r.TypeName.String == "vector" {
				isVector = true
				if matches := vectorDimsRe.FindStringSubmatch(r.DataType.String); len(matches) > 1 {
					if dim, err := strconv.Atoi(matches[1]); err == nil {
						dimensions = dim
					}
				}
			}

			table.Columns = append(table.Columns, ColumnInfo{
				ColumnName:       r.ColumnName.String,
				DataType:         r.DataType.String,
				IsNullable:       r.IsNullable.String,
				Description:      r.ColumnDesc,
				IsPrimaryKey:     r.IsPrimaryKey,
				IsUnique:         r.IsUnique,
				ForeignKeyRef:    r.FKReference,
				IsIndexed:        r.IsIndexed,
				IsIdentity:       r.IdentityType,
				DefaultValue:     r.DefaultValue,
				IsVectorColumn:   isVector,
				VectorDimensions: dimensions,
			})
			columnCount++
		}

		metadata[key] = table
	}

	return metadata, schemas, columnCount
}
