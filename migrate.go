package main

import (
	"flag"
	"fmt"
	"github.com/cloudflare/cfssl/certdb"
	"github.com/fatih/structs"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"           // import to support Postgres
	_ "github.com/mattn/go-sqlite3" // import to support sqlite3
	"log"
	"reflect"
	"strings"
)

func main() {
	sqliteDb := flag.String("sqlite", "./certs.db", "source sqlite db")
	pgConn := flag.String("pg", "postgres://postgres@localhost/postgres", "target postgres db connection string")
	flag.Parse()

	db, err := sqlx.Connect("sqlite3", *sqliteDb)
	if err != nil {
		log.Fatalln(err)
	}

	pg, err := sqlx.Connect("postgres", *pgConn)
	// pg, err := sqlx.Connect("sqlite3", "./test.db")
	if err != nil {
		log.Fatalln(err)
	}

	copyTable(db, pg, "certificates", certdb.CertificateRecord{})
	copyTable(db, pg, "ocsp_responses", certdb.OCSPRecord{})
}

func copyTable(from *sqlx.DB, to *sqlx.DB, table string, record interface{}) {
	// Get the type of the passed in record instance and construct a slice of
	// that type
	vr := reflect.ValueOf(record)
	vcerts := reflect.MakeSlice(reflect.SliceOf(vr.Type()), 0, 0)

	// Create an addressable (pointer) slice of the above by creating a value of
	// that type signature and settings it value to the above reflected slice
	certs := reflect.New(vcerts.Type())
	certs.Elem().Set(vcerts)

	// The interface is certs is now the underlying value, which is &[]record{}
	err := from.Select(certs.Interface(), "SELECT * FROM "+table)
	if err != nil {
		log.Fatalln(err)
	}

	insertSQL := constructInsertSQL(table, record)
	// Reacquire the slice because it will have gone through reallocation to grow
	// to hold the found rows and thus will no longer be the original pointer in
	// memory
	vcerts = certs.Elem()
	copied := 0
	for i := 0; i < vcerts.Len(); i += 1 {
		cert := vcerts.Index(i).Interface()
		res, err := to.NamedExec(insertSQL, cert)
		if err != nil {
			log.Println(err)
		} else {
			inserted, _ := res.RowsAffected()
			if inserted == 0 {
				log.Println("Failed to insert row")
			} else {
				copied++
			}
		}
	}

	log.Printf("Copied %v rows to %v table", copied, table)
}

func constructInsertSQL(table string, s interface{}) string {
	record := structs.New(s)

	var fields []string
	for _, field := range record.Fields() {
		tag := field.Tag("db")
		if tag != "" {
			fields = append(fields, tag)
		}
	}

	return fmt.Sprintf("INSERT INTO %v(%v) VALUES(%v)", table, strings.Join(fields, ", "), ":"+strings.Join(fields, ", :"))
}
