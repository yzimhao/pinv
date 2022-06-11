package ymfile

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type response struct {
	Ok     bool        `json:"ok"`
	Reason string      `json:"reason"`
	Data   interface{} `json:"data"`
}

type FileInfo struct {
	Name         string `json:"name"`
	IsDir        bool   `json:"is_dir"`
	Size         string `json:"size"`
	LastModified int64  `json:"last_modified"`
	ContentType  string `json:"content_type"`
}

func Run() {
	// http.Handle("/", http.FileServer(http.Dir(opt.Dir)))
	// http.ListenAndServe(":8080", nil)
	startGin()
}

func startGin() {
	route := gin.Default()
	gin.SetMode(gin.DebugMode)

	route.LoadHTMLGlob("./*.html")

	route.GET("/", index)
	apiV1 := route.Group("/api/v1")
	{
		apiV1.GET("/main", softMain)
		apiV1.GET("/filelist", fileList)
		apiV1.POST("/upload", upload)
		apiV1.GET("/download", download)
		apiV1.POST("/createdir", createDir)
		apiV1.POST("/delete", delete)
	}
	route.Run(":8080")
}

func index(c *gin.Context) {
	if gin.Mode() == gin.DebugMode {
		c.HTML(http.StatusOK, "index.html", nil)
	} else {
		c.Data(http.StatusOK, "text/html", []byte(indexHtmlString))
	}
}

func softMain(c *gin.Context) {
	c.JSON(http.StatusOK, &response{Ok: true, Reason: "", Data: gin.H{
		"app_name":    GloOptions.AppName(),
		"app_version": GloOptions.Version(),
	}})
}

func createDir(c *gin.Context) {
	newDirName := c.PostForm("dirname")
	curDir := c.PostForm("current_path")

	dirpath := GloOptions.RootDir + "/" + curDir + "/" + newDirName
	logrus.Infof("create dir name: %s", dirpath)

	err := os.MkdirAll(dirpath, os.ModeDir)

	if err != nil {
		c.JSON(http.StatusOK, &response{Ok: false, Reason: err.Error(), Data: nil})
	}
	c.JSON(http.StatusOK, &response{Ok: true, Reason: "", Data: nil})
}

func delete(c *gin.Context) {
	curPath := c.PostForm("current_path")
	fileNames := c.PostFormArray("filename[]")

	for _, item := range fileNames {
		if item == "" || item == ".." || item == "." {
			continue
		}
		fullPath := fmt.Sprintf("%s/%s/%s", GloOptions.RootDir, curPath, item)
		//todo exist
		logrus.Info("delete file ", item, fullPath)
		os.Remove(fullPath)
	}
	c.JSON(http.StatusOK, &response{Ok: true, Reason: "", Data: nil})
}

func upload(c *gin.Context) {
	curPath := c.Query("current_path")
	file, _ := c.FormFile("file")
	if curPath == "" {
		c.JSON(http.StatusOK, &response{Ok: false, Reason: "current path error"})
		return
	}

	savePath := GloOptions.RootDir + curPath + "/" + file.Filename
	err := c.SaveUploadedFile(file, savePath)
	if err != nil {
		logrus.Errorf("upload file error: %v", err)
		c.JSON(http.StatusOK, &response{Ok: false, Reason: "upload file error: %v"})
		return
	}

	logrus.Infof("upload file success: %s", savePath)
	c.JSON(http.StatusOK, &response{Ok: true, Reason: "", Data: nil})
}

func download(c *gin.Context) {
	curPath := c.Query("current_path")
	file := c.Query("filename")

	real := GloOptions.RootDir + curPath + file
	fileContent, _ := ioutil.ReadFile(real)
	//todo 是否存在
	contentType := "application/octet-stream"
	fileContentDisposition := "attachment;filename=\"" + file + "\""
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fileContentDisposition)
	c.Data(http.StatusOK, contentType, fileContent)
}

func fileList(c *gin.Context) {

	current_path := c.Query("current_path")

	if current_path == "" {
		current_path = GloOptions.RootDir
	} else {
		current_path = fmt.Sprintf("%s%s", GloOptions.RootDir, current_path)
	}

	logrus.Info(GloOptions.RootDir, "  ", current_path)

	c.JSON(http.StatusOK, &response{Ok: true, Reason: "", Data: gin.H{"path": current_path, "files": getFiles(current_path)}})
}

func getFiles(pathName string) []FileInfo {
	files := make([]FileInfo, 0)

	logrus.Warn(pathName)
	if pathName != GloOptions.RootDir {
		files = append(files, FileInfo{
			Name:  "..",
			IsDir: true,
		})
	}

	rd, err := ioutil.ReadDir(pathName)
	if err != nil {
		return files
	}

	for _, info := range rd {
		files = append(files, FileInfo{
			Name:         info.Name(),
			IsDir:        info.IsDir(),
			Size:         fileSizeHuman(info.Size()),
			LastModified: info.ModTime().Unix(),
			ContentType:  getContentType(pathName + "/" + info.Name()),
		})
	}
	return files
}

func getContentType(fileName string) string {
	f, err := os.Open(fileName)
	if err != nil {
		return "unknown"
	}
	defer f.Close()

	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil {
		return "unknown"
	}
	return http.DetectContentType(buffer)
}

func fileSizeHuman(fbyte int64) string {
	str := ""
	if fbyte < 1048576 {
		str = fmt.Sprintf("%.0fKB", float64(fbyte/1024))
	} else if fbyte == 1048576 {
		str = "1MB"
	} else if fbyte > 1048576 && fbyte < 1073741824 {
		str = fmt.Sprintf("%.0fMB", float64(fbyte/(1024*1024)))
	} else if fbyte > 1048576 && fbyte == 1073741824 {
		str = "1GB"
	} else if fbyte > 1073741824 && fbyte < 1099511627776 {
		str = fmt.Sprintf("%.0fGB", float64(fbyte/(1024*1024*1024)))
	} else {
		str = ">1TB"
	}
	return str
}
