package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"time"
)

var dbConn *sql.DB
var ipToCountMap map[string]ClientAccess

type ClientAccess struct {
	LastAccess time.Time
	Count int32
}

func main() {
	var err error
	ipToCountMap = make(map[string]ClientAccess)
	dbConn, err = newDBConn()
	if err != nil {
		fmt.Println("error while db setup: ", err)
		return
	}
	defer dbConn.Close()
	r := gin.New()

	r.Use(CORSMiddleware())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})


	r.OPTIONS("/submit")
	r.POST("/submit", SubmitInput)

	//r.RunTLS(":443", "/etc/letsencrypt/live/travel-u.world/fullchain.pem", "/etc/letsencrypt/live/travel-u.world/privkey.pem")
	r.Run(":443")
	//m := autocert.Manager{
	//	Prompt:     autocert.AcceptTOS,
	//	HostPolicy: autocert.HostWhitelist("hammerhead-app-9dvgj.ondigitalocean.app", "localhost"),
	//	Cache:      autocert.DirCache("/var/www/.cache"),
	//}
	//
	//err = autotls.RunWithManager(r, &m)
	//if err != nil {
	//	fmt.Println("error autotls run ", err)
	//}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

type TripInputRequest struct {
	Name string
	Phone string
}

type TripResponseMessage string

func SubmitInput(c *gin.Context) {
	ip := c.ClientIP()
	val, ok := ipToCountMap[ip]
	oneHourBefore := time.Now().Second() - 3600
	if ok && (val.LastAccess.Second() > oneHourBefore && val.Count >= 5) {
		fmt.Println("blacklist IP for 1 hour ", ipToCountMap[ip])
		c.JSON(400, TripResponseMessage("Bad Request Received"))
		return
	}
	if !ok {
		ipToCountMap[ip] = ClientAccess{
			LastAccess: time.Now(),
			Count:      1,
		}
	} else {
		ipToCountMap[ip] = ClientAccess{
			LastAccess: time.Now(),
			Count:      val.Count + 1,
		}
	}

	tripReq  := &TripInputRequest{}
	err := c.BindJSON(tripReq)
	if err != nil {
		fmt.Println("error: ", err)
		c.JSON(400, TripResponseMessage("Bad Request Received"))
		return
	}

	query := "INSERT INTO `user_input` (`name`, `trip_date`, `phone`) VALUES (?, NOW(), ?)"
	insertResult, err := dbConn.ExecContext(c, query, tripReq.Name, tripReq.Phone)
	if err != nil {
		fmt.Println("failed store data ", err)
		c.JSON(500, "failed to take input")
		return
	}
	id, err := insertResult.LastInsertId()
	if err != nil {
		fmt.Println("failed fetch last insert ID ", err)
		c.JSON(500, "failed to take input")
		return
	}
	log.Printf("inserted id: %d", id)

	c.JSON(200, TripResponseMessage("input recorded successfully"))
	return
}

func newDBConn() (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		"doadmin",
		"AVNS_aJLPFtOUYdtOMcT9K-I",
		"db-mysql-blr1-61699-do-user-10640117-0.b.db.ondigitalocean.com",
		25060,
		"trip_inputs",
		)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("error occurred", err)
		return nil, err
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	return db, nil
}