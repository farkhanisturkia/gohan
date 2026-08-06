package gohan

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

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