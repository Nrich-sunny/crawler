package sqldb

/** 本模块是一个更加底层的模块，只进行数据的存储
**	使用了原生的 MySQL 语句来与数据库交互
 */

import (
	"database/sql"
	"errors"
	"go.uber.org/zap"
	"strings"
)

// DBer 数据库的接口
type DBer interface {
	CreateTable(t TableMetaData) error
	Insert(t TableMetaData) error
}

// Sqldb : DBer 的实现
type Sqldb struct {
	options
	db *sql.DB
}

func New(opts ...Option) (*Sqldb, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	d := &Sqldb{}
	d.options = options
	if err := d.OpenDB(); err != nil {
		return nil, err
	}
	return d, nil
}

//func newSqldb() *Sqldb {
//	return &Sqldb{}
//}

// OpenDB 用于与数据库建立连接，需要从外部传入远程 MySQL 数据库的连接地址
func (d *Sqldb) OpenDB() error {
	db, err := sql.Open("mysql", d.sqlUrl)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(2048)
	db.SetMaxIdleConns(2048)
	if err = db.Ping(); err != nil {
		return err
	}
	d.db = db
	return nil
}

type Field struct {
	Title string // 字段名
	Type  string // 字段属性(类型)
}

type TableMetaData struct {
	TableName   string
	ColumnNames []Field       // 标题字段
	Args        []interface{} // 要插入的数据
	DataCount   int           // 插入数据的数量
	AutoKey     bool          // 标识是否为表创建自增主键
}

// CreateTable 拼接 MySQL 语句， 执行数据库操作
func (d *Sqldb) CreateTable(t TableMetaData) error {
	if len(t.ColumnNames) == 0 {
		return errors.New("column can not be empty")
	}

	sql := `CREATE TABLE IF NOT EXISTS ` + t.TableName + " ("
	if t.AutoKey {
		sql += `id INT(12) NOT NULL PRIMARY KEY AUTO_INCREMENT,`
	}
	for _, t := range t.ColumnNames {
		sql += t.Title + ` ` + t.Type + `,`
	}
	sql = sql[:len(sql)-1] + `) ENGINE=MyISAM DEFAULT CHARSET=utf8;`

	d.logger.Debug("crate table", zap.String("sql", sql))

	_, err := d.db.Exec(sql)
	return err
}

func (d *Sqldb) Insert(t TableMetaData) error {
	if len(t.ColumnNames) == 0 {
		return errors.New("empty columns")
	}

	sql := `INSERT INTO ` + t.TableName + `(`
	for _, v := range t.ColumnNames {
		sql += v.Title + ","
	}
	sql = sql[:len(sql)-1] + `) VALUES `
	blank := ",(" + strings.Repeat(",?", len(t.ColumnNames))[1:] + ")"
	sql += strings.Repeat(blank, t.DataCount)[1:] + `;`

	d.logger.Debug("insert table", zap.String("sql", sql))

	_, err := d.db.Exec(sql, t.Args...)
	return err
}
