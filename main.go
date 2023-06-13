package main

import (
	"database/sql"
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
	create_date time.Time
	extension   string
	size        string
}

type FileContent struct {
	ID         int
	Content    []byte
	FileInfoID int
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
		log.Print(err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}

	files, err := getFileByID(intID)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
	}

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

	var (
		fileExist bool
		file      File
	)
	err := db.QueryRow("SELECT CASE COUNT(1) WHEN 0 THEN FALSE ELSE TRUE END FROM file_info WHERE id = ?", id).Scan(&fileExist)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}

	if !fileExist {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	err = c.Bind(&file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File not found"})
		return
	}

	query := "UPDATE file_info SET name = ?, create_date = ?, extension = ?, size = ? WHERE id = ?"
	_, err = db.Exec(query, file.name, file.create_date, file.extension, file.size, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File info updated successfully"})
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
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Upload file error: %s", err.Error())})
		return
	}

	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		log.Print(err)
		return
	}
	defer fileContent.Close()

	fileSize := file.Size
	fileExtension := filepath.Ext(file.Filename)

	fileInfoData := File{
		name:        file.Filename,
		create_date: time.Now().UTC(),
		extension:   fileExtension,
		size:        fmt.Sprintf("%d", fileSize),
	}

	fileInfoStmt, err := db.Prepare("INSERT INTO file_info(name, create_date, extension, size) VALUES (?, ?, ?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		log.Print(err)
		return
	}
	defer fileInfoStmt.Close()

	fileInfoResult, err := fileInfoStmt.Exec(fileInfoData.name, fileInfoData.create_date, fileInfoData.extension, fileInfoData.size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		log.Print(err)
		return
	}

	lastFileInfoInsertID, err := fileInfoResult.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		log.Print(err)
		return
	}

	fileContentBytes, err := ioutil.ReadAll(fileContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		log.Print(err)
		return
	}

	fileContentStmt, err := db.Prepare("INSERT INTO file_content(content, file_info_id) VALUES (?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		log.Print(err)
		return
	}
	defer fileContentStmt.Close()

	_, err = fileContentStmt.Exec(fileContentBytes, lastFileInfoInsertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		log.Print(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File was uploaded"})
}

func GetFiles(c *gin.Context) {
	files, err := getFilesFromDB()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, files)
}
