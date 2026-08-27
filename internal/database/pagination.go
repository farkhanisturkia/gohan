package database

import (
	"fmt"
	"reflect"
	"math"
)

type Pagination struct {
	Total       int64       `json:"total"`
	PerPage     int         `json:"per_page"`
	CurrentPage int         `json:"current_page"`
	LastPage    int         `json:"last_page"`
	Data        interface{} `json:"data"`
}

type PaginateStmt struct {
	c         Commander
	relations []string
	page      int
	perPage   int
}

func (db *DB) Paginate(page, perPage int) *PaginateStmt {
	return &PaginateStmt{c: db, page: page, perPage: perPage}
}

func (tx *Tx) Paginate(page, perPage int) *PaginateStmt {
	return &PaginateStmt{c: tx, page: page, perPage: perPage}
}

func (p *PaginateStmt) Preload(relations ...string) *PaginateStmt {
	p.relations = append(p.relations, relations...)
	return p
}

func (p *PaginateStmt) Find(dest interface{}) (*Pagination, error) {
	if p.page <= 0 {
		p.page = 1
	}
	if p.perPage <= 0 {
		p.perPage = 10
	}

	offset := (p.page - 1) * p.perPage

	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return nil, fmt.Errorf("[error] Paginate.Find requires a pointer to a slice")
	}

	elemType := val.Elem().Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	tableName := getTableName(elemType)
	var total int64

	tableQuotedTable, err := quoteIdentifier(tableName, p.c.driver())
	if err != nil {
		return nil, err
	}
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableQuotedTable)
	if err := p.c.execer().QueryRow(countQuery).Scan(&total); err != nil {
		return nil, err
	}

	limitQuery := fmt.Sprintf("LIMIT %d OFFSET %d", p.perPage, offset)
	if err := findAllWithQuery(p.c, dest, limitQuery, nil); err != nil {
		return nil, err
	}

	if len(p.relations) > 0 {
		if err := executePreload(p.c, dest, p.relations); err != nil {
			return nil, err
		}
	}

	lastPage := int(math.Ceil(float64(total) / float64(p.perPage)))

	return &Pagination{
		Total:       total,
		PerPage:     p.perPage,
		CurrentPage: p.page,
		LastPage:    lastPage,
		Data:        dest,
	}, nil
}