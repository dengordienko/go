package main

import (
	"database/sql"
	"fmt"
	"github.com/mailru/go-clickhouse"
	"log"
	"strconv"
	"strings"
	"time"
)

//connect to DB and prepare table
func connectToCH()  (*sql.DB) {
	// Open opens a database specified by its database driver name and a
	// driver-specific data source name, usually consisting of at least a
	// database name and connection information.
	connect, err := sql.Open("clickhouse", "http://127.0.0.1:8123/default")
	if err != nil {
		log.Fatal(clickhouse.QueryID, err)
	}
	// Ping verifies a connection to the database is still alive,
	// establishing a connection if necessary.
	if err := connect.Ping(); err != nil {
		log.Fatal(err)
	}
	//create table
	_, err = connect.Exec(`
		CREATE TABLE IF NOT EXISTS s1ap_stat (
			dt			DateTime,
			SourceIP	String,
			DestIP		String,
			MME			UInt8,
			eNB			UInt8,
			TAC			UInt8,
			PacketSize	UInt8
		) engine=MergeTree PARTITION BY toYYYYMMDD(dt) ORDER BY (dt, MME, eNB, TAC) SETTINGS index_granularity=8192
	`)
	if err != nil {
		log.Fatal(err)
	}
	//clear table
	_, err = connect.Exec(`
		ALTER TABLE s1ap_stat DELETE WHERE 1
	`)
	if err != nil {
		log.Fatal(err)
	}
	//starting insert routine
	go insertToDB(connect)
	//
	return connect
}
//insert routine
func insertToDB(pDB *sql.DB){
	for {
		qToDBSync.Lock()
		//
		size := len(qToDB)
		if size > 0 {
			start := time.Now()
			//starts transaction
			tx, err := pDB.Begin()
			if err != nil {
				log.Fatal(err)
			}
			// Prepare creates a prepared statement for use within a transaction.
			//
			// The returned statement operates within the transaction and can no longer
			// be used once the transaction has been committed or rolled back.
			stmt, err := tx.Prepare(`
				INSERT INTO s1ap_stat (
					dt,
					SourceIP,
					DestIP,
					MME,
					eNB,
					TAC,
					PacketSize
				) VALUES (
					?, ?, ?, ?, ?, ?, ?
			)`)
			if err != nil {
				log.Fatal(err)
			}
			// Exec executes a prepared statement with the given arguments and
			// returns a Result summarizing the effect of the statement.
			for i := 0; i < size; i++ {
				if _, err := stmt.Exec(
					time.Now(),
					qToDB[i].SourceIP,
					qToDB[i].DestinationIP,
					qToDB[i].MME,
					qToDB[i].ENB,
					qToDB[i].TAC,
					qToDB[i].PacketSize,
				); err != nil {
					log.Fatal(err)
				}
			}
			// Commit commits the transaction
			if err := tx.Commit(); err != nil {
				log.Fatal(err)
			}
			//
			qToDB = qToDB[:0]
			finish := time.Since(start).Seconds()
			log.Printf("Processing %d records, time - %.2fsec...\r\n", size, finish)
		}
		qToDBSync.Unlock()
		//
		if err := pDB.Ping(); err != nil {
			log.Fatal(err)
		}
		//
		time.Sleep(1 * time.Second)
	}
}
//
func selectbyTACFromDB(pDB *sql.DB, TAC uint8) []string {
	//SELECT DISTINCT eNB FROM s1ap_stat WHERE TAC=10
	rows, err := pDB.Query(`SELECT DISTINCT	eNB	FROM s1ap_stat WHERE TAC=` + strconv.Itoa(int(TAC)))
	if err != nil {
		log.Fatal(err)
	}
	// Next prepares the next result row for reading with the Scan method. It
	// returns true on success, or false if there is no next result row or an error
	// happened while preparing it. Err should be consulted to distinguish between
	// the two cases.
	// Every call to Scan, even the first one, must be preceded by a call to Next.
	var result []string
	for rows.Next() {
		var eNB  uint8
		// Scan copies the columns in the current row into the values pointed
		// at by dest. The number of values in dest must be the same as the
		// number of columns in Rows.
		if err := rows.Scan(
			&eNB,
		); err != nil {
			log.Fatal(err)
		}
		result = append(result, "{eNB:" + strconv.Itoa(int(eNB)) + "}")
	}
	return result
}
//
func selectbyMMEFromDB(pDB *sql.DB, MME uint8) []string {
	//SELECT DISTINCT eNB FROM s1ap_stat WHERE MME=1
	rows, err := pDB.Query(`SELECT DISTINCT	eNB	FROM s1ap_stat WHERE MME=` + strconv.Itoa(int(MME)))
	if err != nil {
		log.Fatal(err)
	}
	//
	var result []string
	for rows.Next() {
		var eNB  uint8
		//
		if err := rows.Scan(
			&eNB,
		); err != nil {
			log.Fatal(err)
		}
		result = append(result, "{eNB:" + strconv.Itoa(int(eNB)) + "}")
	}
	return result
}
//
func selectForDrawFromDB(pDB *sql.DB) ([]string, []string) {
	//SELECT DISTINCT eNB,TAC,MME FROM s1ap_stat
	rows, err := pDB.Query(`SELECT DISTINCT eNB,TAC,MME FROM s1ap_stat`)
	if err != nil {
		log.Fatal(err)
	}
	//
	var nodes, links []string
	var name1, name2 string
	var tmp [256]uint8

	for rows.Next() {
		var eNB,TAC,MME  uint8
		//
		if err := rows.Scan(
			&eNB,
			&TAC,
			&MME,
		); err != nil {
			log.Fatal(err)
		}
		//{ "name": "Ma1", "group": 1, "color": 1 },
		name1 = fmt.Sprintf("TAC=%d, eNB=%d", TAC, eNB)
		nodes = append(nodes, "{name:" + name1 + " ,group:1}")
		if tmp[MME] == 0{
			name2 = fmt.Sprintf("MME=%d", MME)
			nodes = append(nodes, "{name:" + name2 + " ,group:1}")
			tmp[MME] = 1
		}
		//{ "source": name1, "target": name2}
		links = append(links, "{source:" + name1 + ", target:" + name2 + "}")
	}
	return nodes, links
}
//
func selectSimplexFromDB(pDB *sql.DB) []string {
	//It can be solved by clear SQL, like SELECT DISTINCT * FROM s1ap_stat WHERE SourceIP NOT IN (SELECT DISTINCT DestinationIP FROMs1ap_stat), but...
	//SELECT DISTINCT SourceIP,DestIP FROM s1ap_stat
	rows, err := pDB.Query(`SELECT DISTINCT SourceIP,DestIP FROM s1ap_stat`)
	if err != nil {
		log.Fatal(err)
	}
	//
	var result []string
	var pairs map[string]string		//make map to find simplexes
	pairs = make(map[string]string)

	for rows.Next() {
		var SrcIP,DstIP  string
		//
		if err := rows.Scan(
			&SrcIP,
			&DstIP,
		); err != nil {
			log.Fatal(err)
		}
		//jjj - separator
		pairs[SrcIP + "jjj" + DstIP] = DstIP + "jjj" + SrcIP
	}
	//try to find pair in map by value
	for k, v := range pairs {
		_, ok := pairs[v]
		if !ok {//if not finded - simplex detected
			slice := strings.Split(k, "jjj")
			result = append(result, "{SourceIP:" + slice[0] + ", DestinationIP:" + slice[1] + "}")
		}
	}
	return result
}
/*
SELECT DISTINCT UniqIP
FROM
(
    SELECT DISTINCT SourceIP AS UniqIP
    FROM s1ap_stat
    UNION ALL
    SELECT DISTINCT DestIP AS UnicIP
    FROM s1ap_stat
)
 */

/*SELECT DISTINCT eNB FROM s1ap_stat WHERE TAC=10

FROM
(
SELECT DISTINCT eNB AS UniqIP

UNION ALL
SELECT DISTINCT DestIP AS UnicIP
FROM s1ap_stat
)
*/