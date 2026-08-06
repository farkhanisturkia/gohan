package gohan

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	Driver string
}

func GetConn(e *Env) (*DB, error) {
	var dsn string
	driver := e.DBDriver

	switch driver {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			e.DBUser, e.DBPassword, e.DBHost, e.DBPort, e.DBName)

	case "postgres", "postgresql":
		driver = "postgres"
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			e.DBHost, e.DBPort, e.DBUser, e.DBPassword, e.DBName)

	case "sqlite", "sqlite3":
		driver = "sqlite3"
		dsn = e.DBFile

	default:
		return nil, fmt.Errorf("[error] The '%s' driver is not supported", e.DBDriver)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("[error] Failed to initialize the %s driver: %w", driver, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("[error] Failed to connect to the %s database: %w", driver, err)
	}

	log.Printf("[info] Successfully connected to the database (%s)\n", driver)

	return &DB{
		DB:     db,
		Driver: driver,
	}, nil
}

func (db *DB) SetTable(model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("[error] SetTable requires a struct or a pointer to a struct")
	}

	typ := val.Type()
	tableName := strings.ToLower(typ.Name()) + "s"

	var columns []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		gohanTag := field.Tag.Get("gohan")

		if gohanTag == "-" {
			continue
		}

		colName := strings.ToLower(field.Name)
		dbType := getSQLType(field.Type.Kind(), db.Driver)
		constraints := ""

		if gohanTag != "" {
			tagParts := strings.Split(gohanTag, ";")
			for _, part := range tagParts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "type:") {
					dbType = strings.TrimPrefix(part, "type:")
				} else if part == "primary_key" {
					if db.Driver == "sqlite3" {
						constraints += " PRIMARY KEY AUTOINCREMENT"
					} else if db.Driver == "mysql" {
						constraints += " PRIMARY KEY AUTO_INCREMENT"
					} else if db.Driver == "postgres" {
						dbType = "SERIAL"
						constraints += " PRIMARY KEY"
					}
				} else if part == "not_null" {
					constraints += " NOT NULL"
				} else if part == "unique" {
					constraints += " UNIQUE"
				}
			}
		}

		colDef := fmt.Sprintf("%s %s%s", colName, dbType, constraints)
		columns = append(columns, colDef)
	}

	var query string
	switch db.Driver {
	case "mysql":
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s);", tableName, strings.Join(columns, ", "))
	case "postgres":
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (%s);", tableName, strings.Join(columns, ", "))
	default:
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, strings.Join(columns, ", "))
	}

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("[error] Failed to create the '%s' table: %w", tableName, err)
	}

	log.Printf("[info] The '%s' table is ready to use (created/verified).\n", tableName)
	return nil
}

func getSQLType(kind reflect.Kind, driver string) string {
	switch kind {
	case reflect.Int, reflect.Int64:
		return "INTEGER"
	case reflect.String:
		return "TEXT"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

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
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("[error] `Create` requires a struct or a pointer to a struct")
	}

	typ := val.Type()
	tableName := strings.ToLower(typ.Name()) + "s"

	var columns []string
	var placeholders []string
	var values []interface{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)
		gohanTag := field.Tag.Get("gohan")

		if gohanTag == "-" {
			continue
		}

		if strings.Contains(gohanTag, "primary_key") && (fieldValue.Kind() == reflect.Int || fieldValue.Kind() == reflect.Int64) && fieldValue.Int() == 0 {
			continue
		}

		colName := strings.ToLower(field.Name)
		columns = append(columns, colName)

		if db.Driver == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)+1))
		} else {
			placeholders = append(placeholders, "?")
		}

		values = append(values, fieldValue.Interface())
	}

	var query string
	switch db.Driver {
	case "mysql":
		query = fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", db.quoteIdentifier(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	case "postgres":
		query = fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s)", db.quoteIdentifier(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	default:
		query = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", db.quoteIdentifier(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	}

	_, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("[error] Failed to insert data into '%s': %w", tableName, err)
	}

	log.Printf("[info] The data was successfully saved to the '%s' table\n", tableName)
	return nil
}

func (db *DB) Update(model interface{}, id interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	structType := val.Type()
	tableName := strings.ToLower(structType.Name()) + "s"

	var setClauses []string
	var values []interface{}
	paramIdx := 1

	pkCol := "id"
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := val.Field(i)
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

func (db *DB) quoteIdentifier(name string) string {
	switch db.Driver {
	case "mysql":
		return fmt.Sprintf("`%s`", name)
	case "postgres":
		return fmt.Sprintf("\"%s\"", name)
	default:
		return name
	}
}

func (db *DB) getColumns(tableName string) ([]string, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", db.quoteIdentifier(tableName))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rows.Columns()
}