package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

type RawQuery struct {
	c     Commander
	query string
	args  []interface{}
}

func (r *RawQuery) Scan(dest interface{}) error {
	query := r.query
	if r.c.driver() == "postgres" {
		query = formatPostgresPlaceholders(query)
	}

	rows, err := r.c.execer().Query(query, r.args...)
	if err != nil {
		return fmt.Errorf("[error] Raw query failed: %w", err)
	}
	defer rows.Close()

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("[error] Scan destination must be a pointer")
	}

	elem := destVal.Elem()

	if elem.Kind() == reflect.Slice {
		return scanSlice(rows, elem)
	}

	if elem.Kind() == reflect.Struct {
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return err
			}
			return fmt.Errorf("[warning] Data not found")
		}
		return scanStruct(rows, elem)
	}

	if rows.Next() {
		return rows.Scan(dest)
	}

	return fmt.Errorf("[warning] Data not found")
}

func scanStruct(rows *sql.Rows, structVal reflect.Value) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	structType := structVal.Type()
	fieldMap := buildFieldMap(structType, structVal)

	scanArgs := make([]interface{}, len(cols))
	for i, colName := range cols {
		if v, ok := fieldMap[strings.ToLower(colName)]; ok && v.CanAddr() {
			scanArgs[i] = v.Addr().Interface()
		} else {
			var dummy interface{}
			scanArgs[i] = &dummy
		}
	}

	return rows.Scan(scanArgs...)
}

func scanSlice(rows *sql.Rows, sliceVal reflect.Value) error {
	structType := sliceVal.Type().Elem()
	isPtr := structType.Kind() == reflect.Ptr
	if isPtr {
		structType = structType.Elem()
	}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		elemPtr := reflect.New(structType)
		elem := elemPtr.Elem()

		fieldMap := buildFieldMap(structType, elem)

		scanArgs := make([]interface{}, len(cols))
		for i, colName := range cols {
			if val, ok := fieldMap[strings.ToLower(colName)]; ok && val.CanAddr() {
				scanArgs[i] = val.Addr().Interface()
			} else {
				var dummy interface{}
				scanArgs[i] = &dummy
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		if isPtr {
			sliceVal.Set(reflect.Append(sliceVal, elemPtr))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, elem))
		}
	}

	return rows.Err()
}

func buildFieldMap(structType reflect.Type, structVal reflect.Value) map[string]reflect.Value {
	fieldMap := make(map[string]reflect.Value)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		val := structVal.Field(i)

		if !field.IsExported() || field.Tag.Get("gohan") == "-" {
			continue
		}

		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			nestedMap := buildFieldMap(field.Type, val)
			for k, v := range nestedMap {
				fieldMap[k] = v
			}
			continue
		}

		col := getColumnName(field)
		fieldMap[strings.ToLower(col)] = val
	}

	return fieldMap
}