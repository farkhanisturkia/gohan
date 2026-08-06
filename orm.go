package gohan

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"
)

func (db *DB) FindAll(dest interface{}) error {
	sliceVal := reflect.ValueOf(dest)
	if sliceVal.Kind() != reflect.Ptr || sliceVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("[error] FindAll requires a pointer to a slice of structs")
	}

	sliceElem := sliceVal.Elem()
	structType := sliceElem.Type().Elem()
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	tableName := strings.ToLower(structType.Name()) + "s"
	query := fmt.Sprintf("SELECT * FROM %s", db.quoteIdentifier(tableName))

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("[error] FindAll query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		elemPtr := reflect.New(structType)
		elem := elemPtr.Elem()

		fieldMap := make(map[string]reflect.Value)
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.Tag.Get("gohan") != "-" {
				fieldMap[strings.ToLower(field.Name)] = elem.Field(i)
			}
		}

		scanArgs := make([]interface{}, len(cols))
		for i, colName := range cols {
			if val, ok := fieldMap[strings.ToLower(colName)]; ok {
				scanArgs[i] = val.Addr().Interface()
			} else {
				var dummy interface{}
				scanArgs[i] = &dummy
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		sliceElem.Set(reflect.Append(sliceElem, elem))
	}

	return nil
}

func (db *DB) FindByID(dest interface{}, id interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] FindByID requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := strings.ToLower(structType.Name()) + "s"

	pkCol := "id"
	for i := 0; i < structType.NumField(); i++ {
		tag := structType.Field(i).Tag.Get("gohan")
		if strings.Contains(tag, "primary_key") {
			pkCol = strings.ToLower(structType.Field(i).Name)
			break
		}
	}

	placeholder := "?"
	if db.Driver == "postgres" {
		placeholder = "$1"
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = %s LIMIT 1", db.quoteIdentifier(tableName), db.quoteIdentifier(pkCol), placeholder)
	row := db.QueryRow(query, id)

	cols, err := db.getColumns(tableName)
	if err != nil {
		return err
	}

	fieldMap := make(map[string]reflect.Value)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.Tag.Get("gohan") != "-" {
			fieldMap[strings.ToLower(field.Name)] = structVal.Field(i)
		}
	}

	scanArgs := make([]interface{}, len(cols))
	for i, colName := range cols {
		if v, ok := fieldMap[strings.ToLower(colName)]; ok {
			scanArgs[i] = v.Addr().Interface()
		} else {
			var dummy interface{}
			scanArgs[i] = &dummy
		}
	}

	if err := row.Scan(scanArgs...); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("[warning] Data not found")
		}
		return err
	}

	return nil
}

func (db *DB) Create(model interface{}) error {
	val := reflect.ValueOf(model)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] `Create` requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := strings.ToLower(structType.Name()) + "s"

	var columns []string
	var placeholders []string
	var values []interface{}

	var pkField reflect.Value
	var pkColName string
	var hasPK bool

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structVal.Field(i)
		gohanTag := field.Tag.Get("gohan")

		if gohanTag == "-" {
			continue
		}

		colName := strings.ToLower(field.Name)

		if strings.Contains(gohanTag, "primary_key") {
			hasPK = true
			pkColName = colName
			pkField = fieldValue

			if (fieldValue.Kind() == reflect.Int || fieldValue.Kind() == reflect.Int64) && fieldValue.Int() == 0 {
				continue
			}
		}

		columns = append(columns, db.quoteIdentifier(colName))

		if db.Driver == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)+1))
		} else {
			placeholders = append(placeholders, "?")
		}

		values = append(values, fieldValue.Interface())
	}

	quotedTable := db.quoteIdentifier(tableName)

	if db.Driver == "postgres" && hasPK {
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
			quotedTable,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
			db.quoteIdentifier(pkColName),
		)

		var lastInsertID int64
		err := db.QueryRow(query, values...).Scan(&lastInsertID)
		if err != nil {
			return fmt.Errorf("[error] Failed to insert data into '%s': %w", tableName, err)
		}

		if pkField.IsValid() && pkField.CanSet() {
			pkField.SetInt(lastInsertID)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			quotedTable,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)

		res, err := db.Exec(query, values...)
		if err != nil {
			return fmt.Errorf("[error] Failed to insert data into '%s': %w", tableName, err)
		}

		if hasPK && pkField.IsValid() && pkField.CanSet() {
			if lastID, err := res.LastInsertId(); err == nil && lastID > 0 {
				pkField.SetInt(lastID)
			}
		}
	}

	log.Printf("[info] The data was successfully saved to the '%s' table\n", tableName)
	return nil
}

func (db *DB) Update(model interface{}, id interface{}) error {
	val := reflect.ValueOf(model)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] `Update` requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := strings.ToLower(structType.Name()) + "s"

	var setClauses []string
	var values []interface{}
	paramIdx := 1

	pkCol := "id"
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)
		gohanTag := field.Tag.Get("gohan")

		if gohanTag == "-" {
			continue
		}

		colName := strings.ToLower(field.Name)

		if strings.Contains(gohanTag, "primary_key") {
			pkCol = colName
			continue
		}

		placeholder := "?"
		if db.Driver == "postgres" {
			placeholder = fmt.Sprintf("$%d", paramIdx)
			paramIdx++
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = %s", db.quoteIdentifier(colName), placeholder))
		values = append(values, fieldVal.Interface())
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("[warning] No columns were updated")
	}

	pkPlaceholder := "?"
	if db.Driver == "postgres" {
		pkPlaceholder = fmt.Sprintf("$%d", paramIdx)
	}
	values = append(values, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		db.quoteIdentifier(tableName),
		strings.Join(setClauses, ", "),
		db.quoteIdentifier(pkCol),
		pkPlaceholder,
	)

	res, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("[error] Data update failed: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("[warning] Data not found or no changes")
	}

	log.Printf("[info] The data in the '%s' table with ID %v has been successfully updated\n", tableName, id)
	return nil
}

func (db *DB) Delete(model interface{}, id interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	structType := val.Type()
	tableName := strings.ToLower(structType.Name()) + "s"

	pkCol := "id"
	for i := 0; i < structType.NumField(); i++ {
		if strings.Contains(structType.Field(i).Tag.Get("gohan"), "primary_key") {
			pkCol = strings.ToLower(structType.Field(i).Name)
			break
		}
	}

	placeholder := "?"
	if db.Driver == "postgres" {
		placeholder = "$1"
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s", db.quoteIdentifier(tableName), db.quoteIdentifier(pkCol), placeholder)
	res, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("[error] Failed to delete data: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("[warning] Data not found")
	}

	log.Printf("[info] The data in the '%s' table with ID %v was successfully deleted\n", tableName, id)
	return nil
}