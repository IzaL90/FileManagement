package main

import (
	"database/sql"
	b64 "encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	//"path/filepath"
	"strconv"

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

type FileContent struct{
	ID          int
    Content     []byte
    FileInfoID  int


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
	if err != nil {
		panic(err.Error())
	}

	router := gin.Default()
	router.GET("/files", GetFiles)
	//fmt.Println(getFileByID(1))
	router.GET("/files/:id", GetFileById)
	router.POST("/file", PostFile)
	router.PUT("/files/:id", updateFile)
	router.DELETE("/files/:id", deleteFile)

	router.Run("localhost:8080")
}
func GetFileById(c *gin.Context) {
	id := c.Param("id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(intID)
	}
	log.Print(err)

	files, err := getFileByID(intID)

	if files.id == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "id not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, files)

}
func getAllFiles(c *gin.Context) {

	files, err := getFilesFromDB()
	if err != nil {
		fmt.Println(err)
	}
	if len(files) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "file not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, files)
}

func getFileByID(id int) (File, error) {
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
	file.name = "kicia"

	if err := c.ShouldBindJSON(&file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	query := "UPDATE file_info SET name = ?, extension = ?, size = ? WHERE id = ?"
	_, err := db.Exec(query, file.name, file.extension, file.size, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}
	
	fmt.Println(err.Error())
	c.JSON(http.StatusOK, gin.H{"message": "File updated successfully"})
	log.Print(file)
}

func deleteFile(c *gin.Context) {

	var fileExist bool
	id := c.Param("id")
	err := db.QueryRow("Select case count(1) when 0 then false else true end FROM file_info WHERE id=?", id).Scan(&fileExist)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Something went wrong"})
		return

	}
	fmt.Println(err)

	
	if fileExist {
		_, err = db.Exec("DELETE FROM file_info WHERE id=?", id)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Something went wrong"})
			return

		}
		_, err = db.Exec("DELETE FROM file_content WHERE file_info_id=?", id)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Something went wrong"})
			return

		}

		c.IndentedJSON(http.StatusOK, gin.H{"message": "No content"})
		
	} else {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "file does not exist"})
	}
	
}

func PostFile(c *gin.Context) {
	var (
		file       File
		//extensionn = filepath.Ext(file.name)
		//namee      = file.name[0 : len(file.name)-len(extensionn)]
		//fileContent FileContent
	)
	data := "go"
	sEnc := b64.StdEncoding.EncodeToString([]byte(data))
	fmt.Println(sEnc)
	sDec, _ := b64.StdEncoding.DecodeString(sEnc)
	fmt.Println(string(sDec))
	uEnc := b64.URLEncoding.EncodeToString([]byte(data))

	fileInfoStmt, err := db.Prepare("INSERT INTO file_info(name, create_date, extension, size) VALUES (?, ?, ?, ?)")
	if err != nil {
		fmt.Println(err)
	}
	defer fileInfoStmt.Close()
	fileInfoResult, err := fileInfoStmt.Exec(file.name, file.create_date, file.extension, file.size)
	if err != nil {
		fmt.Println(err)
	}
	lastFileInfoInsertId, err := fileInfoResult.LastInsertId()
	if err != nil {
		fmt.Println(err)
	}

	
	fileContentStmt, err := db.Prepare("INSERT INTO file_content(content, file_info_id) VALUES (?, ?)")
	if err != nil {
		fmt.Println(err)
	}
	defer fileContentStmt.Close()
	fileContentResult, err := fileContentStmt.Exec(uEnc, lastFileInfoInsertId)
	if err != nil {
		fmt.Println(err)
	}
	lastFileContentInsertId, err := fileContentResult.LastInsertId()
	if err != nil {
		fmt.Println(err)
	}

	c.IndentedJSON(http.StatusFound, "/files/"+strconv.Itoa(int(lastFileContentInsertId)))
}


func GetFiles(c *gin.Context) {
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
}
