package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/ochronus/instapaper-go-client/instapaper"

	"github.com/anaskhan96/soup"
)

type instapaperConfig struct {
	ClientID     string `envconfig:"INSTAPAPER_CLIENT_ID"`
	ClientSecret string `envconfig:"INSTAPAPER_CLIENT_SECRET"`
	Username     string `envconfig:"INSTAPAPER_USERNAME"`
	Password     string `envconfig:"INSTAPAPER_PASSWORD"`
	MediumFolder string `envconfig:"INSTAPAPER_MEDIUM_FOLDER" default:"Medium"`
}

type mediumConfig struct {
	ZipName string `envconfig:"MEDIUM_ZIP_NAME" default:"Medium"`
}

var instapaperCfg instapaperConfig
var mediumCfg mediumConfig
var instapaperClient instapaper.Client

func init() {
	loadInstapaperConfig()

	loadMediumConfig()

	initInstapaperClient()
}

func loadInstapaperConfig() {
	if err := envconfig.Process("INSTAPAPER_", &instapaperCfg); err != nil {
		log.Fatalf("error loading instapaper config: %v", err)
	}
}

func loadMediumConfig() {
	if err := envconfig.Process("MEDIUM_", &mediumCfg); err != nil {
		log.Fatalf("error loading Medium config: %v", err)
	}
}

func initInstapaperClient() {
	client, err := newInstapaperClient()
	if err != nil {
		log.Fatalf("error creating instapaper client: %v", err)
	}

	instapaperClient = client
	if err := instapaperClient.Authenticate(); err != nil {
		log.Fatalf("error authenticate instapaper client: %v", err)
	}
}

func newInstapaperClient() (instapaper.Client, error) {
	return instapaper.NewClient(instapaperCfg.ClientID, instapaperCfg.ClientSecret, instapaperCfg.Username, instapaperCfg.Password)
}

func getFolderSvc() instapaper.FolderService {
	return instapaper.FolderService{Client: instapaperClient}
}

func getBookmarkSvc() instapaper.BookmarkService {
	return instapaper.BookmarkService{Client: instapaperClient}
}

func createFolder() error {
	log.Println("start to create folder for medium in instapaper")

	fldr, err := getFolderByTitle(instapaperCfg.MediumFolder)
	if err != nil {
		return err
	}

	if len(fldr.ID) == 0 {
		svc := getFolderSvc()

		_, err := svc.Add(instapaperCfg.MediumFolder)
		if err != nil {
			return err
		}
	}

	log.Println("finished to create folder for medium in instapaper")
	return nil
}

func getFolderByTitle(title string) (instapaper.Folder, error) {
	svc := getFolderSvc()

	folderList, err := svc.List()
	if err != nil {
		return instapaper.Folder{}, err
	}

	for _, folder := range folderList {
		if folder.Title == title {
			return folder, nil
		}
	}

	return instapaper.Folder{}, nil
}

type operations struct {
	succeedOperations []succeedOperation
	failedOperations  []failedOperation
}

type succeedOperation struct {
	article savedArticle
}

type failedOperation struct {
	article savedArticle
	err     error
}

func (op operations) saveAddResultToCsv() {
	op.saveSucceedOpsToCsv()
	op.saveFailedOpsToCsv()
}

func (op operations) saveFailedOpsToCsv() {
	csvFile, err := os.Create("failed.csv")

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvFile)

	_ = csvwriter.Write([]string{"url", "title", "error_reason"})
	for _, failedOp := range op.failedOperations {
		_ = csvwriter.Write([]string{failedOp.article.url, failedOp.article.title, failedOp.err.Error()})
	}

	csvwriter.Flush()
	csvFile.Close()
}

func (op operations) saveSucceedOpsToCsv() {
	csvFile, err := os.Create("succeed.csv")

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvFile)

	_ = csvwriter.Write([]string{"url", "title"})
	for _, succeedOp := range op.succeedOperations {
		_ = csvwriter.Write([]string{succeedOp.article.url, succeedOp.article.title})
	}

	csvwriter.Flush()
	csvFile.Close()
}

func addToInstapaper(articles []savedArticle) operations {
	bkmrkSvc := getBookmarkSvc()

	succeedOps := make([]succeedOperation, 0)
	failedOps := make([]failedOperation, 0)

	log.Println("start to add saved articles in medium to instapaper")

	for i, article := range articles {
		log.Println(fmt.Sprintf("trying to add the-%d articles: %s (%s)", i, article.title, article.url))

		_, err := bkmrkSvc.Add(instapaper.BookmarkAddRequestParams{
			URL:         article.url,
			Title:       article.title,
			Folder:      instapaperCfg.MediumFolder,
			Description: "Hi! this article is added by using github.com/irfansofyana/medium-to-instapaper",
		})

		if err != nil {
			failedOps = append(failedOps, failedOperation{
				article: article,
				err:     err,
			})

			continue
		}

		succeedOps = append(succeedOps, succeedOperation{
			article: article,
		})

		log.Println(fmt.Sprintf("finished trying to add the-%d articles: %s (%s)", i, article.title, article.url))
	}

	log.Println("finished to add all saved articles in medium to instapaper")

	return operations{succeedOperations: succeedOps, failedOperations: failedOps}
}

type myCloser interface {
	Close() error
}

func closeFile(f myCloser) {
	err := f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func readAll(file *zip.File) []byte {
	fc, err := file.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer closeFile(fc)

	content, err := ioutil.ReadAll(fc)
	if err != nil {
		log.Fatal(err)
	}

	return content
}

type savedArticle struct {
	title string
	url   string
}

func getArticlesFromBookmark() ([]savedArticle, error) {
	savedArticles := make([]savedArticle, 0)

	zf, err := zip.OpenReader(mediumCfg.ZipName)
	if err != nil {
		fmt.Println(mediumCfg.ZipName)
		return savedArticles, err
	}

	defer closeFile(zf)

	for _, file := range zf.File {
		if strings.HasPrefix(file.Name, "bookmarks") {
			content := readAll(file)
			savedArticles = append(savedArticles, extractArticles(string(content))...)
		}
	}

	return savedArticles, nil
}

func extractArticles(htmlBookmark string) []savedArticle {
	doc := soup.HTMLParse(htmlBookmark)

	savedArticles := make([]savedArticle, 0)

	lists := doc.FindAll("ul")
	for _, list := range lists {
		links := list.FindAll("a")
		for _, link := range links {
			savedArticles = append(savedArticles, savedArticle{
				title: link.Text(),
				url:   link.Attrs()["href"],
			})
		}
	}

	return savedArticles
}

func main() {
	if err := createFolder(); err != nil {
		log.Fatalf("error creating folder in instapaper: %v", err)
	}

	savedArticles, err := getArticlesFromBookmark()
	if err != nil {
		log.Fatalf("error get articles from medium bookmark: %v", err)
	}

	ops := addToInstapaper(savedArticles)
	ops.saveAddResultToCsv()
}
