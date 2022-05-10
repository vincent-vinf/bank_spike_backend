package db

import (
	"bank_spike_backend/internal/orm"
	"bank_spike_backend/internal/util/config"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
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

func Login(phone, passwd string, isAdmin bool) (int, error) {
	db := getInstance()
	query := "select id from users where phone = ? and passwd = ?"
	if isAdmin {
		query = "select id from admin where phone = ? and passwd = ?"
	}
	stmt, err := db.Prepare(query)
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

func GetUserById(id string) (*orm.User, error) {
	db := getInstance()
	stmt, err := db.Prepare("select username,phone,id_number,work_status,age from users where id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	u := &orm.User{}
	u.ID = id
	if rows.Next() {
		err := rows.Scan(&u.Username, &u.Phone, &u.IDNumber, &u.WorkStatus, &u.Age)
		if err != nil {
			return nil, err
		}
	}
	return u, nil
}

func GetSpikeById(id string) (*orm.Spike, error) {
	db := getInstance()
	stmt, err := db.Prepare("select commodity_id,quantity,access_rule,start_time,end_time from spike where id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	s := &orm.Spike{}
	s.ID = id
	if rows.Next() {
		var cId int
		err := rows.Scan(&cId, &s.Quantity, &s.AccessRule, &s.StartTime, &s.EndTime)
		if err != nil {
			return nil, err
		}
		s.CommodityID = strconv.Itoa(cId)
	}
	return s, nil
}

func GetActiveSpike() ([]*orm.Spike, error) {
	db := getInstance()
	stmt, err := db.Prepare("select commodity_id,quantity,access_rule,start_time,end_time from spike where start_time <= ? and end_time >= ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	now := time.Now()
	rows, err := stmt.Query(now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*orm.Spike
	for rows.Next() {
		var cId int
		s := &orm.Spike{}

		err := rows.Scan(&cId, &s.Quantity, &s.AccessRule, &s.StartTime, &s.EndTime)
		if err != nil {
			return nil, err
		}
		s.CommodityID = strconv.Itoa(cId)

		res = append(res, s)
	}
	return res, nil
}

func GetSleepSpike() ([]*orm.Spike, error) {
	db := getInstance()
	stmt, err := db.Prepare("select commodity_id,quantity,access_rule,start_time,end_time from spike where start_time > ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	now := time.Now()
	rows, err := stmt.Query(now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*orm.Spike
	for rows.Next() {
		var cId int
		s := &orm.Spike{}

		err := rows.Scan(&cId, &s.Quantity, &s.AccessRule, &s.StartTime, &s.EndTime)
		if err != nil {
			return nil, err
		}
		s.CommodityID = strconv.Itoa(cId)

		res = append(res, s)
	}
	return res, nil
}
