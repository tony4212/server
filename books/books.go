package books

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	cError "gitlab.com/joyread/ultimate/error"
	"gitlab.com/joyread/ultimate/models"
)

func hasPrefix(opSplit []string, content string) string {
	for _, element := range opSplit {
		if strings.HasPrefix(element, content) {
			return strings.Trim(strings.Split(element, ":")[1], " ")
		}
	}
	return ""
}

// UploadBooks ...
func UploadBooks(c *gin.Context) {
	form, err := c.MultipartForm()
	cError.CheckError(err)
	files := form.File["upload[]"]

	for _, file := range files {
		fileStoragePath := fmt.Sprintf("./uploads/%s", file.Filename)
		if err = c.SaveUploadedFile(file, fileStoragePath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": fmt.Sprintf("upload file err: %s", err.Error()),
			})
			return
		} else {
			cmd := exec.Command("./xpdf-tools-win-4.00/bin64/pdfinfo.exe", fileStoragePath)

			var out bytes.Buffer
			cmd.Stdout = &out

			err := cmd.Run()
			cError.CheckError(err)

			output := out.String()
			opSplit := strings.Split(output, "\n")

			// Get book title.
			title := hasPrefix(opSplit, "Title")

			// Get author of the uploaded book.
			author := hasPrefix(opSplit, "Author")

			// Get total number of pages.
			pages := hasPrefix(opSplit, "Pages")

			coverPath := fmt.Sprintf("./uploads/img/%s", file.Filename)
			fmt.Println(coverPath)

			cmd = exec.Command("./xpdf-tools-win-4.00/bin64/pdftopng.exe", "-f", "1", "-l", "1", fileStoragePath, coverPath)

			err = cmd.Run()
			cError.CheckError(err)

			if _, err := os.Stat(coverPath + "-000001.png"); err == nil {
				coverPath = "/cover/" + file.Filename + "-000001.png"
			} else {
				cError.CheckError(err)
			}

			token, _ := c.Cookie("joyread-token")

			db, ok := c.MustGet("db").(*sql.DB)
			if !ok {
				fmt.Println("Middleware db error")
			}

			accountID := models.GetUserIDFromToken(db, token)
			filePath := fmt.Sprintf("/book/%s", file.Filename)

			booksModel := models.BooksModel{
				Title:     title,
				Author:    author,
				Pages:     pages,
				FilePath:  filePath,
				CoverPath: coverPath,
				AccountID: accountID,
			}

			models.InsertBooks(db, booksModel)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "Successfully uploaded",
	})
}

// GetBook
func GetBook(c *gin.Context) {
	bookName := c.Param("bookName")
	userPresent, ok := c.MustGet("userPresent").(bool)
	if !ok {
		fmt.Println("Middleware user error")
	}

	if userPresent {
		c.HTML(http.StatusOK, "pdf-wrapper.html", gin.H{
			"bookName": bookName,
		})
	} else {
		c.Redirect(http.StatusMovedPermanently, "/signup")
	}
}

// Viewer ...
func Viewer(c *gin.Context) {
	userPresent, ok := c.MustGet("userPresent").(bool)
	if !ok {
		fmt.Println("Middleware user error")
	}

	if userPresent {
		c.HTML(http.StatusOK, "pdf-viewer.html", "")
	} else {
		c.Redirect(http.StatusMovedPermanently, "/signup")
	}
}
