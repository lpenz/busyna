package libbusyna

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func dbOpen(dbfilename string) *sql.DB {
	// Connect/create database file
	db, err := sql.Open("sqlite3", dbfilename)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("PRAGMA synchronous = OFF")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("PRAGMA journal_mode = OFF")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

// DbWrite dumps the CmdData that it reads from the provided channel into a sqlite database.
func DbWrite(c <-chan CmdData, dbfilename string) {
	os.Remove(dbfilename)

	// Connect/create database file
	db := dbOpen(dbfilename)

	// Create tables:
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Command ( id INTEGER PRIMARY KEY, line VARCHAR(255) NOT NULL, env VARCHAR(1023), dir VARCHAR(255) )")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Dep ( cmdid INTEGER, filename VARCHAR(255) NOT NULL )")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Target ( cmdid INTEGER, filename VARCHAR(255) NOT NULL )")
	if err != nil {
		log.Fatal(err)
	}

	// Insert data
	cmdIns, err := db.Prepare("INSERT INTO Command(line, env, dir) values(?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	depIns, err := db.Prepare("INSERT INTO Dep(cmdid, filename) values(?,?)")
	if err != nil {
		log.Fatal(err)
	}
	targetIns, err := db.Prepare("INSERT INTO Target(cmdid, filename) values(?,?)")
	if err != nil {
		log.Fatal(err)
	}
	for cmddata := range c {
		env, err := json.Marshal(cmddata.Cmd.Env)
		if err != nil {
			log.Fatal(err)
		}
		res, err := cmdIns.Exec(cmddata.Cmd.Line, string(env), cmddata.Cmd.Dir)
		if err != nil {
			log.Fatal(err)
		}
		cmdid, err := res.LastInsertId()
		if err != nil {
			log.Fatal(err)
		}
		for dep := range cmddata.Deps {
			_, err = depIns.Exec(cmdid, dep)
			if err != nil {
				log.Fatal(err)
			}
		}
		for target := range cmddata.Targets {
			_, err = targetIns.Exec(cmdid, target)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	db.Close()
}

// DbRead reads the database from filename and outputs the data to the returned
// channel.
func DbRead(dbfilename string) <-chan CmdData {
	rChan := make(chan CmdData)

	// Connect/create database file
	db := dbOpen(dbfilename)

	go func() {
		defer close(rChan)
		defer db.Close()

		// Get commands:
		cmdrows, err := db.Query("SELECT * FROM Command")
		if err != nil {
			log.Fatal(err)
		}

		// Iterate them:
		for cmdrows.Next() {
			var cmdid int
			var line string
			var envstr string
			var dir string
			if err := cmdrows.Scan(&cmdid, &line, &envstr, &dir); err != nil {
				log.Fatal(err)
			}
			var env map[string]string
			if err = json.Unmarshal([]byte(envstr), &env); err != nil {
				log.Fatal(err)
			}
			cmd := Cmd{line, env, dir}

			rChan <- CmdData{
				cmd,
				dbReadSet(db, "Dep", cmdid),
				dbReadSet(db, "Target", cmdid),
			}
		}
	}()
	return rChan
}

// dbReadSet reads deps or targets from the database.
func dbReadSet(db *sql.DB, tablename string, cmdid int) map[string]bool {
	q := fmt.Sprintf("SELECT filename FROM %s WHERE cmdid = ?", tablename)
	rows, err := db.Query(q, cmdid)
	if err != nil {
		log.Fatal(err)
	}
	ret := map[string]bool{}
	for rows.Next() {
		var val string
		if err := rows.Scan(&val); err != nil {
			log.Fatal(err)
		}
		ret[val] = true
	}
	return ret
}
