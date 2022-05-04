package db

import (
	"bank_spike_backend/internal/util/config"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"sync"
	"time"
)

var (
	db   *sql.DB
	once sync.Once
)

func getInstance() *sql.DB {
	if db == nil {
		once.Do(func() {
			cfg := config.GetConfig().Mysql
			source := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.User, cfg.Passwd, cfg.Host, cfg.Port, cfg.Database)
			var err error
			db, err = sql.Open("mysql", source)
			if err != nil {
				panic(err)
			}
			db.SetConnMaxLifetime(time.Minute * 3)
			db.SetMaxOpenConns(10)
			db.SetMaxIdleConns(10)
		})
	}
	return db
}

func Close() {
	if db != nil {
		_ = db.Close()
	}
}

func Register(username, phone, passwd string) (int, error) {
	db := getInstance()
	stmt, err := db.Prepare("insert into users (username , phone, passwd) VALUES (?,?,?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	res, err := stmt.Exec(username, phone, passwd)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func Login(phone, passwd string) (int, error) {
	db := getInstance()
	stmt, err := db.Prepare("select id from users where phone = ? and passwd = ?")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(phone, passwd)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	return 0, errors.New("does not exist")
}

func IsExistPhone(phone string) (bool, error) {
	db := getInstance()
	stmt, err := db.Prepare("select id from users where phone = ?")
	if err != nil {
		return false, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(phone)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if rows.Next() {
		return true, nil
	}
	return false, nil
}