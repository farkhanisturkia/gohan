package database

import (
	"fmt"
	"reflect"
	"strings"
)

type PreloadStmt struct {
	c         Commander
	relations []string
}

func (db *DB) Preload(relations ...string) *PreloadStmt {
	return &PreloadStmt{c: db, relations: relations}
}

func (tx *Tx) Preload(relations ...string) *PreloadStmt {
	return &PreloadStmt{c: tx, relations: relations}
}

func (p *PreloadStmt) FindAll(dest interface{}) error {
	if err := findAll(p.c, dest); err != nil {
		return err
	}
	return executePreload(p.c, dest, p.relations)
}

func (p *PreloadStmt) FindOne(dest interface{}, query string, args ...interface{}) error {
	if err := findOne(p.c, dest, query, args...); err != nil {
		return err
	}
	return executePreload(p.c, dest, p.relations)
}

func executePreload(c Commander, dest interface{}, relations []string) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("[error] executePreload requires a non-nil pointer")
	}

	elemVal := val.Elem()
	var sliceVal reflect.Value
	isSingleStruct := false

	if elemVal.Kind() == reflect.Struct {
		isSingleStruct = true
		tmpSlice := reflect.MakeSlice(reflect.SliceOf(elemVal.Type()), 1, 1)
		tmpSlice.Index(0).Set(elemVal)
		sliceVal = tmpSlice
	} else if elemVal.Kind() == reflect.Slice {
		sliceVal = elemVal
	} else {
		return fmt.Errorf("[error] executePreload requires a pointer to a struct or slice")
	}

	if sliceVal.Len() == 0 {
		return nil
	}

	firstItem := reflect.Indirect(sliceVal.Index(0))
	if !firstItem.IsValid() || firstItem.Kind() != reflect.Struct {
		return nil
	}

	parentType := firstItem.Type()
	parentPKName := getPrimaryKeyColumn(parentType)

	for _, relName := range relations {
		sampleItem := reflect.Indirect(sliceVal.Index(0))
		relField := sampleItem.FieldByName(relName)
		if !relField.IsValid() {
			continue
		}

		relType := relField.Type()
		isHasMany := relType.Kind() == reflect.Slice
		isPtr := false

		if isHasMany {
			relType = relType.Elem()
		}
		if relType.Kind() == reflect.Ptr {
			relType = relType.Elem()
			isPtr = true
		}

		if isHasMany {
			if err := preloadHasMany(c, sliceVal, parentType, parentPKName, relName, relType, relField.Type(), isPtr); err != nil {
				return err
			}
		} else {
			if err := preloadBelongsTo(c, sliceVal, relName, relType, isPtr); err != nil {
				return err
			}
		}
	}

	if isSingleStruct && elemVal.CanSet() {
		elemVal.Set(sliceVal.Index(0))
	}

	return nil
}

func preloadHasMany(c Commander, sliceVal reflect.Value, parentType reflect.Type, parentPKName, relName string, relType, relFieldType reflect.Type, isPtr bool) error {
	fkColName := toSnakeCase(parentType.Name()) + "_id"
	var parentIDs []interface{}
	parentIDMap := make(map[int64]bool)

	for i := 0; i < sliceVal.Len(); i++ {
		itemVal := reflect.Indirect(sliceVal.Index(i))
		if !itemVal.IsValid() {
			continue
		}

		pkField := itemVal.FieldByName(toCamelCase(parentPKName))
		if !pkField.IsValid() {
			continue
		}

		pid, ok := extractInt64(pkField)
		if ok && pid > 0 && !parentIDMap[pid] {
			parentIDMap[pid] = true
			parentIDs = append(parentIDs, pid)
		}
	}

	if len(parentIDs) == 0 {
		return nil
	}

	tableName := getTableName(relType)
	query, err := buildInQuery(c, tableName, fkColName, len(parentIDs))
	if err != nil {
		return err
	}

	rows, err := c.execer().Query(query, parentIDs...)
	if err != nil {
		return fmt.Errorf("[preload error] failed querying %s: %w", relName, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	loadedMap := make(map[int64]reflect.Value)
	for rows.Next() {
		elemPtr := reflect.New(relType)
		elem := elemPtr.Elem()

		scanArgs := make([]interface{}, len(cols))
		fieldMap := make(map[string]reflect.Value)
		for i := 0; i < relType.NumField(); i++ {
			f := relType.Field(i)
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
			return fmt.Errorf("[preload error] failed scanning %s: %w", relName, err)
		}

		var fkValue int64
		for i := 0; i < relType.NumField(); i++ {
			f := relType.Field(i)
			if getColumnName(f) == fkColName {
				fkValue, _ = extractInt64(elem.Field(i))
				break
			}
		}

		sliceValue, exists := loadedMap[fkValue]
		if !exists {
			sliceValue = reflect.MakeSlice(relFieldType, 0, 0)
		}

		if isPtr {
			sliceValue = reflect.Append(sliceValue, elemPtr)
		} else {
			sliceValue = reflect.Append(sliceValue, elem)
		}
		loadedMap[fkValue] = sliceValue
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("[preload error] row iteration error on %s: %w", relName, err)
	}

	for i := 0; i < sliceVal.Len(); i++ {
		itemVal := reflect.Indirect(sliceVal.Index(i))
		if !itemVal.IsValid() {
			continue
		}

		pkField := itemVal.FieldByName(toCamelCase(parentPKName))
		targetField := itemVal.FieldByName(relName)

		if pkField.IsValid() && targetField.IsValid() && targetField.CanSet() {
			pid, _ := extractInt64(pkField)
			if relSlice, found := loadedMap[pid]; found {
				targetField.Set(relSlice)
			}
		}
	}

	return nil
}

func preloadBelongsTo(c Commander, sliceVal reflect.Value, relName string, relType reflect.Type, isPtr bool) error {
	var fkIDs []interface{}
	idMap := make(map[int64]bool)

	for i := 0; i < sliceVal.Len(); i++ {
		itemVal := reflect.Indirect(sliceVal.Index(i))
		if !itemVal.IsValid() {
			continue
		}

		fkField := itemVal.FieldByName(relName + "ID")
		if !fkField.IsValid() {
			continue
		}

		fkID, ok := extractInt64(fkField)
		if ok && fkID > 0 && !idMap[fkID] {
			idMap[fkID] = true
			fkIDs = append(fkIDs, fkID)
		}
	}

	if len(fkIDs) == 0 {
		return nil
	}

	tableName := getTableName(relType)
	pkColName := getPrimaryKeyColumn(relType)

	query, err := buildInQuery(c, tableName, pkColName, len(fkIDs))
	if err != nil {
		return err
	}

	rows, err := c.execer().Query(query, fkIDs...)
	if err != nil {
		return fmt.Errorf("[preload error] failed querying %s: %w", relName, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	loadedRelMap := make(map[int64]reflect.Value)
	for rows.Next() {
		elemPtr := reflect.New(relType)
		elem := elemPtr.Elem()

		scanArgs := make([]interface{}, len(cols))
		fieldMap := make(map[string]reflect.Value)
		for i := 0; i < relType.NumField(); i++ {
			f := relType.Field(i)
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
			return fmt.Errorf("[preload error] failed scanning %s: %w", relName, err)
		}

		var pkID int64
		for i := 0; i < relType.NumField(); i++ {
			f := relType.Field(i)
			if getColumnName(f) == pkColName {
				pkID, _ = extractInt64(elem.Field(i))
				break
			}
		}

		loadedRelMap[pkID] = elem
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("[preload error] row iteration error on %s: %w", relName, err)
	}

	for i := 0; i < sliceVal.Len(); i++ {
		itemVal := reflect.Indirect(sliceVal.Index(i))
		if !itemVal.IsValid() {
			continue
		}

		fkField := itemVal.FieldByName(relName + "ID")
		targetField := itemVal.FieldByName(relName)

		if fkField.IsValid() && targetField.IsValid() && targetField.CanSet() {
			fkID, _ := extractInt64(fkField)
			if relData, found := loadedRelMap[fkID]; found {
				if isPtr {
					ptrVal := reflect.New(relType)
					ptrVal.Elem().Set(relData)
					targetField.Set(ptrVal)
				} else {
					targetField.Set(relData)
				}
			}
		}
	}

	return nil
}

func extractInt64(v reflect.Value) (int64, bool) {
	if !v.IsValid() {
		return 0, false
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint()), true
	default:
		return 0, false
	}
}

func buildInQuery(c Commander, tableName, colName string, count int) (string, error) {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		if c.driver() == "postgres" {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		} else {
			placeholders[i] = "?"
		}
	}

	tableQuoted, err := quoteIdentifier(tableName, c.driver())
	if err != nil {
		return "", err
	}

	colQuoted, err := quoteIdentifier(colName, c.driver())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", tableQuoted, colQuoted, strings.Join(placeholders, ", ")), nil
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}