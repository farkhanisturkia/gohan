package database

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

type Commander interface {
	execer() Execer
	driver() string
}

func (db *DB) Raw(query string, args ...interface{}) *RawQuery {
	return &RawQuery{c: db, query: query, args: args}
}

func (tx *Tx) Raw(query string, args ...interface{}) *RawQuery {
	return &RawQuery{c: tx, query: query, args: args}
}

func (db *DB) FindAll(dest interface{}) error { return findAll(db, dest) }
func (db *DB) FindOne(dest interface{}, condition string, args ...interface{}) error {
	return findOne(db, dest, condition, args...)
}
func (db *DB) FindByID(dest interface{}, id interface{}) error { return findByID(db, dest, id) }
func (db *DB) Create(model interface{}) error                 { return create(db, model) }
func (db *DB) Update(model interface{}, id interface{}) error  { return update(db, model, id) }
func (db *DB) Delete(model interface{}, id interface{}) error  { return deleteModel(db, model, id) }

func (tx *Tx) FindAll(dest interface{}) error { return findAll(tx, dest) }
func (tx *Tx) FindOne(dest interface{}, condition string, args ...interface{}) error {
	return findOne(tx, dest, condition, args...)
}
func (tx *Tx) FindByID(dest interface{}, id interface{}) error { return findByID(tx, dest, id) }
func (tx *Tx) Create(model interface{}) error                 { return create(tx, model) }
func (tx *Tx) Update(model interface{}, id interface{}) error  { return update(tx, model, id) }
func (tx *Tx) Delete(model interface{}, id interface{}) error  { return deleteModel(tx, model, id) }

func findAll(c Commander, dest interface{}) error {
	sliceVal := reflect.ValueOf(dest)
	if sliceVal.Kind() != reflect.Ptr || sliceVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("[error] FindAll requires a pointer to a slice of structs")
	}

	sliceElem := sliceVal.Elem()
	structType := sliceElem.Type().Elem()
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	tableName := toSnakeCase(structType.Name()) + "s"
	query := fmt.Sprintf("SELECT * FROM %s", quoteIdentifier(tableName, c.driver()))

	rows, err := c.execer().Query(query)
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
			if !field.IsExported() || field.Tag.Get("gohan") == "-" {
				continue
			}
			fieldMap[getColumnName(field)] = elem.Field(i)
		}

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
			return fmt.Errorf("[error] Scan failed: %w", err)
		}

		if sliceElem.Type().Elem().Kind() == reflect.Ptr {
			sliceElem.Set(reflect.Append(sliceElem, elemPtr))
		} else {
			sliceElem.Set(reflect.Append(sliceElem, elem))
		}
	}

	return rows.Err()
}

func findOne(c Commander, dest interface{}, condition string, args ...interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] FindOne requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := toSnakeCase(structType.Name()) + "s"

	if c.driver() == "postgres" {
		condition = formatPostgresPlaceholders(condition)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", quoteIdentifier(tableName, c.driver()), condition)
	rows, err := c.execer().Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return fmt.Errorf("[warning] Data not found")
	}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	fieldMap := make(map[string]reflect.Value)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() || field.Tag.Get("gohan") == "-" {
			continue
		}
		fieldMap[getColumnName(field)] = structVal.Field(i)
	}

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

func findByID(c Commander, dest interface{}, id interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] FindByID requires a pointer to a struct")
	}

	structType := val.Elem().Type()
	pkCol := getPrimaryKeyColumn(structType)

	placeholder := "?"
	if c.driver() == "postgres" {
		placeholder = "$1"
	}

	condition := fmt.Sprintf("%s = %s", quoteIdentifier(pkCol, c.driver()), placeholder)
	return findOne(c, dest, condition, id)
}

func create(c Commander, model interface{}) error {
	val := reflect.ValueOf(model)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] `Create` requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := toSnakeCase(structType.Name()) + "s"

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

		if !field.IsExported() || gohanTag == "-" {
			continue
		}

		colName := getColumnName(field)

		if strings.Contains(gohanTag, "primary_key") {
			hasPK = true
			pkColName = colName
			pkField = fieldValue

			if (fieldValue.Kind() == reflect.Int || fieldValue.Kind() == reflect.Int64) && fieldValue.Int() == 0 {
				continue
			}
		}

		columns = append(columns, quoteIdentifier(colName, c.driver()))

		if c.driver() == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)+1))
		} else {
			placeholders = append(placeholders, "?")
		}

		values = append(values, fieldValue.Interface())
	}

	if len(columns) == 0 {
		return fmt.Errorf("[error] No insertable columns found for '%s'", tableName)
	}

	quotedTable := quoteIdentifier(tableName, c.driver())

	if c.driver() == "postgres" && hasPK {
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
			quotedTable,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
			quoteIdentifier(pkColName, c.driver()),
		)

		if pkField.Kind() == reflect.Int || pkField.Kind() == reflect.Int64 {
			var lastInsertID int64
			err := c.execer().QueryRow(query, values...).Scan(&lastInsertID)
			if err != nil {
				return fmt.Errorf("[error] Failed to insert data into '%s': %w", tableName, err)
			}
			if pkField.CanSet() {
				pkField.SetInt(lastInsertID)
			}
		} else {
			_, err := c.execer().Exec(query, values...)
			if err != nil {
				return fmt.Errorf("[error] Failed to insert data into '%s': %w", tableName, err)
			}
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			quotedTable,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)

		res, err := c.execer().Exec(query, values...)
		if err != nil {
			return fmt.Errorf("[error] Failed to insert data into '%s': %w", tableName, err)
		}

		if hasPK && pkField.IsValid() && pkField.CanSet() && (pkField.Kind() == reflect.Int || pkField.Kind() == reflect.Int64) {
			if lastID, err := res.LastInsertId(); err == nil && lastID > 0 {
				pkField.SetInt(lastID)
			}
		}
	}

	log.Printf("[info] The data was successfully saved to the '%s' table\n", tableName)
	return nil
}

func update(c Commander, model interface{}, id interface{}) error {
	val := reflect.ValueOf(model)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] `Update` requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := toSnakeCase(structType.Name()) + "s"

	var setClauses []string
	var values []interface{}
	paramIdx := 1

	pkCol := getPrimaryKeyColumn(structType)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)
		gohanTag := field.Tag.Get("gohan")

		if !field.IsExported() || gohanTag == "-" {
			continue
		}

		colName := getColumnName(field)

		if strings.Contains(gohanTag, "primary_key") {
			continue
		}

		placeholder := "?"
		if c.driver() == "postgres" {
			placeholder = fmt.Sprintf("$%d", paramIdx)
			paramIdx++
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quoteIdentifier(colName, c.driver()), placeholder))
		values = append(values, fieldVal.Interface())
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("[warning] No columns were updated")
	}

	pkPlaceholder := "?"
	if c.driver() == "postgres" {
		pkPlaceholder = fmt.Sprintf("$%d", paramIdx)
	}
	values = append(values, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		quoteIdentifier(tableName, c.driver()),
		strings.Join(setClauses, ", "),
		quoteIdentifier(pkCol, c.driver()),
		pkPlaceholder,
	)

	res, err := c.execer().Exec(query, values...)
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

func deleteModel(c Commander, model interface{}, id interface{}) error {
	t := reflect.TypeOf(model)
	if t == nil {
		return fmt.Errorf("[error] Delete requires a valid struct or struct pointer")
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("[error] Delete requires a struct or a pointer to a struct")
	}

	tableName := toSnakeCase(t.Name()) + "s"
	pkCol := getPrimaryKeyColumn(t)

	placeholder := "?"
	if c.driver() == "postgres" {
		placeholder = "$1"
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s", quoteIdentifier(tableName, c.driver()), quoteIdentifier(pkCol, c.driver()), placeholder)
	res, err := c.execer().Exec(query, id)
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

func quoteIdentifier(name string, driver string) string {
	cleanName := strings.ReplaceAll(name, "`", "")
	cleanName = strings.ReplaceAll(cleanName, "\"", "")

	switch driver {
	case "mysql":
		return fmt.Sprintf("`%s`", cleanName)
	case "postgres":
		return fmt.Sprintf("\"%s\"", cleanName)
	default:
		return cleanName
	}
}

func getPrimaryKeyColumn(t reflect.Type) string {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if strings.Contains(field.Tag.Get("gohan"), "primary_key") {
			return getColumnName(field)
		}
	}
	return "id"
}

func formatPostgresPlaceholders(condition string) string {
	var result strings.Builder
	paramIdx := 1
	inString := false

	for _, char := range condition {
		if char == '\'' {
			inString = !inString
		}
		if char == '?' && !inString {
			result.WriteString(fmt.Sprintf("$%d", paramIdx))
			paramIdx++
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}