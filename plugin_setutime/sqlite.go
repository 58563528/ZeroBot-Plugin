package setutime

import (
	"database/sql"
	"reflect"
	"strings"

	_ "modernc.org/sqlite"
)

// sqlite 数据库对象
type sqlite struct {
	DB     *sql.DB
	DBPath string
}

// create 生成数据库
// 默认结构体的第一个元素为主键
// 返回错误
func (db *sqlite) create(table string, objptr interface{}) (err error) {
	if db.DB == nil {
		database, err := sql.Open("sqlite", db.DBPath)
		if err != nil {
			return err
		}
		db.DB = database
	}
	var (
		tags  = tags(objptr)
		kinds = kinds(objptr)
		top   = len(tags) - 1
		cmd   = []string{}
	)
	cmd = append(cmd, "CREATE TABLE IF NOT EXISTS")
	cmd = append(cmd, table)
	cmd = append(cmd, "(")
	for i := range tags {
		cmd = append(cmd, tags[i])
		cmd = append(cmd, kinds[i])
		switch i {
		default:
			cmd = append(cmd, "NULL,")
		case 0:
			cmd = append(cmd, "PRIMARY KEY")
			cmd = append(cmd, "NOT NULL,")
		case top:
			cmd = append(cmd, "NULL);")
		}
	}
	if _, err := db.DB.Exec(strings.Join(cmd, " ")); err != nil {
		return err
	}
	return nil
}

// insert 插入数据集
// 默认结构体的第一个元素为主键
// 返回错误
func (db *sqlite) insert(table string, objptr interface{}) (err error) {
	rows, err := db.DB.Query("SELECT * FROM " + table)
	if err != nil {
		return err
	}
	tags, _ := rows.Columns()
	rows.Close()
	var (
		values = values(objptr)
		top    = len(tags) - 1
		cmd    = []string{}
	)
	cmd = append(cmd, "INSERT INTO")
	cmd = append(cmd, table)
	for i := range tags {
		switch i {
		default:
			cmd = append(cmd, tags[i])
			cmd = append(cmd, ",")
		case 0:
			cmd = append(cmd, "(")
			cmd = append(cmd, tags[i])
			cmd = append(cmd, ",")
		case top:
			cmd = append(cmd, tags[i])
			cmd = append(cmd, ")")
		}
	}
	for i := range tags {
		switch i {
		default:
			cmd = append(cmd, "?")
			cmd = append(cmd, ",")
		case 0:
			cmd = append(cmd, "VALUES (")
			cmd = append(cmd, "?")
			cmd = append(cmd, ",")
		case top:
			cmd = append(cmd, "?")
			cmd = append(cmd, ")")
		}
	}
	stmt, err := db.DB.Prepare(strings.Join(cmd, " "))
	if err != nil {
		return err
	}
	_, err = stmt.Exec(values...)
	if err != nil {
		return err
	}
	return nil
}

// find 查询数据库
// condition 可为"WHERE id = 0"
// 默认字段与结构体元素顺序一致
// 返回错误
func (db *sqlite) find(table string, objptr interface{}, condition string) (err error) {
	var cmd = []string{}
	cmd = append(cmd, "SELECT * FROM ")
	cmd = append(cmd, table)
	cmd = append(cmd, condition)
	rows, err := db.DB.Query(strings.Join(cmd, " "))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err != nil {
			return err
		}
		err = rows.Scan(addrs(objptr)...)
		if err != nil {
			return err
		}
	}
	return nil
}

// del 删除数据库
// condition 可为"WHERE id = 0"
// 返回错误
func (db *sqlite) del(table string, condition string) (err error) {
	var cmd = []string{}
	cmd = append(cmd, "DELETE FROM")
	cmd = append(cmd, table)
	cmd = append(cmd, condition)
	stmt, err := db.DB.Prepare(strings.Join(cmd, " "))
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	return nil
}

// num 查询数据库行数
// 返回行数以及错误
func (db *sqlite) num(table string) (num int, err error) {
	var cmd = []string{}
	cmd = append(cmd, "SELECT * FROM")
	cmd = append(cmd, table)
	rows, err := db.DB.Query(strings.Join(cmd, " "))
	if err != nil {
		return num, err
	}
	defer rows.Close()
	for rows.Next() {
		num++
	}
	return num, nil
}

// tags 反射 返回结构体对象的 tag 数组
func tags(objptr interface{}) []string {
	var tags []string
	elem := reflect.ValueOf(objptr).Elem()
	// 判断第一个元素是否为匿名字段
	if elem.Type().Field(0).Anonymous {
		elem = elem.Field(0)
	}
	for i, flen := 0, elem.Type().NumField(); i < flen; i++ {
		tags = append(tags, elem.Type().Field(i).Tag.Get("db"))
	}
	return tags
}

// kinds 反射 返回结构体对象的 kinds 数组
func kinds(objptr interface{}) []string {
	var kinds []string
	elem := reflect.ValueOf(objptr).Elem()
	// 判断第一个元素是否为匿名字段
	if elem.Type().Field(0).Anonymous {
		elem = elem.Field(0)
	}
	for i, flen := 0, elem.Type().NumField(); i < flen; i++ {
		switch elem.Field(i).Type().String() {
		case "int64":
			kinds = append(kinds, "INT")
		case "string":
			kinds = append(kinds, "TEXT")
		default:
			kinds = append(kinds, "TEXT")
		}
	}
	return kinds
}

// values 反射 返回结构体对象的 values 数组
func values(objptr interface{}) []interface{} {
	var values []interface{}
	elem := reflect.ValueOf(objptr).Elem()
	// 判断第一个元素是否为匿名字段
	if elem.Type().Field(0).Anonymous {
		elem = elem.Field(0)
	}
	for i, flen := 0, elem.Type().NumField(); i < flen; i++ {
		switch elem.Field(i).Type().String() {
		case "int64":
			values = append(values, elem.Field(i).Int())
		case "string":
			values = append(values, elem.Field(i).String())
		default:
			values = append(values, elem.Field(i).String())
		}
	}
	return values
}

// addrs 反射 返回结构体对象的 addrs 数组
func addrs(objptr interface{}) []interface{} {
	var addrs []interface{}
	elem := reflect.ValueOf(objptr).Elem()
	// 判断第一个元素是否为匿名字段
	if elem.Type().Field(0).Anonymous {
		elem = elem.Field(0)
	}
	for i, flen := 0, elem.Type().NumField(); i < flen; i++ {
		addrs = append(addrs, elem.Field(i).Addr().Interface())
	}
	return addrs
}
