package database

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type Tabler interface {
	TableName() string
}

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

func getTableName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	instance := reflect.New(t).Interface()
	if tabler, ok := instance.(Tabler); ok {
		return tabler.TableName()
	}

	name := toSnakeCase(t.Name())
	if strings.HasSuffix(name, "y") && !strings.HasSuffix(name, "ay") && !strings.HasSuffix(name, "ey") && !strings.HasSuffix(name, "oy") {
		return name[:len(name)-1] + "ies"
	}
	if strings.HasSuffix(name, "s") || strings.HasSuffix(name, "x") || strings.HasSuffix(name, "z") || strings.HasSuffix(name, "ch") || strings.HasSuffix(name, "sh") {
		return name + "es"
	}
	return name + "s"
}

func findAll(c Commander, dest interface{}) error {
	return findAllWithQuery(c, dest, "", nil)
}

func findAllWithQuery(c Commander, dest interface{}, querySuffix string, args []interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("[error] FindAll requires a pointer to a slice")
	}

	sliceVal := val.Elem()
	elemType := sliceVal.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr

	if isPtr {
		elemType = elemType.Elem()
	}

	tableName := getTableName(elemType)
	quotedTable, err := quoteIdentifier(tableName, c.driver())
	if err != nil {
		return err
	}

	rawQuery := fmt.Sprintf("SELECT * FROM %s %s", quotedTable, querySuffix)
	if c.driver() == "postgres" {
		rawQuery = formatPostgresPlaceholders(rawQuery)
	}

	rows, err := c.execer().Query(rawQuery, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		elemPtr := reflect.New(elemType)
		elem := elemPtr.Elem()

		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		scanArgs := make([]interface{}, len(cols))
		fieldMap := make(map[string]reflect.Value)

		for i := 0; i < elemType.NumField(); i++ {
			f := elemType.Field(i)
			if !f.IsExported() || f.Tag.Get("gohan") == "-" {
				continue
			}
			fieldMap[getColumnName(f)] = elem.Field(i)
		}

		for i, col := range cols {
			if v, ok := fieldMap[strings.ToLower(col)]; ok && v.CanAddr() {
				scanArgs[i] = v.Addr().Interface()
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

	return nil
}

func findOne(c Commander, dest interface{}, condition string, args ...interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] FindOne requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := getTableName(structType)
	quotedTable, err := quoteIdentifier(tableName, c.driver())
	if err != nil {
		return err
	}

	if c.driver() == "postgres" {
		condition = formatPostgresPlaceholders(condition)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", quotedTable, condition)
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
	quotedPK, err := quoteIdentifier(pkCol, c.driver())
	if err != nil {
		return err
	}

	placeholder := "?"
	if c.driver() == "postgres" {
		placeholder = "$1"
	}

	condition := fmt.Sprintf("%s = %s", quotedPK, placeholder)
	return findOne(c, dest, condition, id)
}

func create(c Commander, model interface{}) error {
	val := reflect.ValueOf(model)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[error] `Create` requires a pointer to a struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()
	tableName := getTableName(structType)

	var columns []string
	var placeholders []string
	var values []interface{}

	var pkField reflect.Value
	var pkColName string
	var hasPK bool

	now := time.Now()

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

		valInterface := fieldValue.Interface()
		if t, ok := valInterface.(time.Time); ok {
			if t.IsZero() && (field.Name == "CreatedAt" || field.Name == "UpdatedAt") {
				valInterface = now
				if fieldValue.CanSet() {
					fieldValue.Set(reflect.ValueOf(now))
				}
			}
		}

		quotedCol, err := quoteIdentifier(colName, c.driver())
		if err != nil {
			return err
		}
		columns = append(columns, quotedCol)

		if c.driver() == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)+1))
		} else {
			placeholders = append(placeholders, "?")
		}

		values = append(values, valInterface)
	}

	if len(columns) == 0 {
		return fmt.Errorf("[error] No insertable columns found for '%s'", tableName)
	}

	tableQuotedTable, err := quoteIdentifier(tableName, c.driver())
	if err != nil {
		return err
	}

	if c.driver() == "postgres" && hasPK {
		quotedPK, err := quoteIdentifier(pkColName, c.driver())
		if err != nil {
			return err
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
			tableQuotedTable,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
			quotedPK,
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
			tableQuotedTable,
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
	tableName := getTableName(structType)

	var setClauses []string
	var values []interface{}
	paramIdx := 1

	pkCol := getPrimaryKeyColumn(structType)
	now := time.Now()

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

		valInterface := fieldVal.Interface()
		if field.Name == "UpdatedAt" {
			if _, ok := valInterface.(time.Time); ok {
				valInterface = now
				if fieldVal.CanSet() {
					fieldVal.Set(reflect.ValueOf(now))
				}
			}
		}

		placeholder := "?"
		if c.driver() == "postgres" {
			placeholder = fmt.Sprintf("$%d", paramIdx)
			paramIdx++
		}

		quotedTable, err := quoteIdentifier(colName, c.driver())
		if err != nil {
			return err
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedTable, placeholder))
		values = append(values, valInterface)
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("[warning] No columns were updated")
	}

	pkPlaceholder := "?"
	if c.driver() == "postgres" {
		pkPlaceholder = fmt.Sprintf("$%d", paramIdx)
	}
	values = append(values, id)

	tableQuotedTable, err := quoteIdentifier(tableName, c.driver())
	if err != nil {
		return err
	}

	pkQuotedTable, err := quoteIdentifier(pkCol, c.driver())
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		tableQuotedTable,
		strings.Join(setClauses, ", "),
		pkQuotedTable,
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

	tableName := getTableName(t)
	pkCol := getPrimaryKeyColumn(t)

	placeholder := "?"
	if c.driver() == "postgres" {
		placeholder = "$1"
	}

	tableQuotedTable, err := quoteIdentifier(tableName, c.driver())
	if err != nil {
		return err
	}

	pkQuotedTable, err := quoteIdentifier(pkCol, c.driver())
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s", tableQuotedTable, pkQuotedTable, placeholder)
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

func quoteIdentifier(name string, driver string) (string, error) {
	cleanName := strings.ReplaceAll(name, "`", "")
	cleanName = strings.ReplaceAll(cleanName, "\"", "")

	if !validIdentifier.MatchString(cleanName) {
		return "", fmt.Errorf("[security error] invalid SQL identifier: '%s'", name)
	}

	switch driver {
	case "mysql":
		return fmt.Sprintf("`%s`", cleanName), nil
	case "postgres":
		return fmt.Sprintf("\"%s\"", cleanName), nil
	default:
		return cleanName, nil
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