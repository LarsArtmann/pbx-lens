package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", os.Args[1])
	if err != nil {
		panic(err)
	}

	var n int
	if err := db.QueryRow(`select count(*) from meta_stream_log`).Scan(&n); err != nil {
		panic(err)
	}

	fmt.Println("events:", n)
}
