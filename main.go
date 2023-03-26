package main

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	b64 "encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	
)

type File struct {
	id          int
	name        string
	create_date []uint8
	extension   string
	size        string
}

func getDbConnection() (*sql.DB, error) {
	var db *sql.DB
	var err error
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "files",
		AllowNativePasswords: false,
	}
	_ = godotenv.Load("pass.env")
	
	secretKey := os.Getenv("SECRET_KEY")
	dsn := "root:" + secretKey + "@tcp(127.0.0.1:3306)/files"
	db, err = sql.Open("mysql", cfg.FormatDSN())
	dsn = dsn + "?allowNativePasswords=false"
	log.Println("using", dsn)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	return db, err
}

var db *sql.DB

func getFilesFromDB() ([]File, error) {
	var files []File = make([]File, 0, 0)
	rows, err := db.Query("SELECT id, name FROM file_info")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f File
		err = rows.Scan(&f.id, &f.name)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}
func main() {
	var err error
	db, err = getDbConnection()
	file, err := os.Open("hej.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*file)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	data := "go"
	sEnc := b64.StdEncoding.EncodeToString([]byte(data))
	fmt.Println(sEnc)
	sDec, _ := b64.StdEncoding.DecodeString(sEnc)
	fmt.Println(string(sDec))
	fmt.Println()
	uEnc := b64.URLEncoding.EncodeToString([]byte(data))
	fmt.Println(uEnc)
	uDec, _ := b64.URLEncoding.DecodeString(uEnc)
	fmt.Println(string(uDec))
	name := uEnc
	create_date := time.Now()
	extension := ".txt"
	size := len(uEnc)
	fmt.Println(name, create_date, extension, size)
	
	var id int
	var nam string
	var content string
	err = db.QueryRow("SELECT file_info.id, file_info.name, file_content.content "+
		"FROM file_info "+
		"INNER JOIN file_content ON file_info.id = file_content.file_info_id "+
		"WHERE file_info.id = ?", 1).Scan(&id, &nam, &content)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File id: %d\nFile name: %s\nFile content: %s\n", id, nam, content)
	decodedContent, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(nam, decodedContent, 0644)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File %s downloaded successfully\n", nam)
	filee, err := os.Open(nam)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*filee)
	router := gin.Default()
	router.GET("/files", func(c *gin.Context) {
		files, err := getFilesFromDB()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if len(files) == 0 {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "file not found"})
			return
		}
		c.IndentedJSON(http.StatusOK, files)
	})
	fmt.Println(filesByID(1))
	router.GET("/files/:id", GetId)
	router.POST("/file", func(c *gin.Context) {
		var file File
		var extensionn = filepath.Ext(file.name)
		var namee = file.name[0 : len(file.name)-len(extensionn)]
		stmt, err := db.Prepare("INSERT INTO file_content(content, file_info_id) VALUES (?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		result, err := stmt.Exec(uEnc, id, namee)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(result)
		lastInsertId, err := result.LastInsertId()
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(lastInsertId)
		id := lastInsertId

		c.Redirect(http.StatusFound, "/files/"+strconv.Itoa(int(id)))
	})
	router.PUT("/files/:id", updateFile)
	router.DELETE("/files/:id", deleteFile)

	router.Run("localhost:8080")
}
func GetId(c *gin.Context) {
	id := c.Param("id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(intID)
	}
	log.Print(err)

	files, err := filesByID(intID)

	if files.id == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "id not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, files)

}
func getAllFiles(c *gin.Context) {
	_, err := getDbConnection()
	//fmt.Println(db)

	files, err := getFilesFromDB()
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "file not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, files)
}

func filesByID(id int) (File, error) {
	var err error
	var file File

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File found: %v\n", id)

	row := db.QueryRow("SELECT * from file_info where id = ?", id)
	if err := row.Scan(&file.id, &file.name, &file.create_date, &file.extension, &file.size); err != nil {
		if err == sql.ErrNoRows {
			return file, fmt.Errorf("filesById %d: no such file", id)
		}
		return file, fmt.Errorf("filesById %d: %v", id, err)

	}
	log.Print(err)
	return file, nil

}

func updateFile(c *gin.Context) {
	id := c.Param("id")
	var file File
	if err := c.ShouldBindJSON(&file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := "UPDATE file_info SET name = ?, extension = ?, size = ? WHERE id = ?"
	_, err := db.Exec(query, file.name, file.extension, file.size, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "File updated successfully"})
}

func deleteFile(c *gin.Context) {

	id := c.Param("id")
	c.Redirect(http.StatusFound, "/files/")
	_, a := db.Exec("DELETE file_info WHERE id = ?", id)
	fmt.Println(a)

}