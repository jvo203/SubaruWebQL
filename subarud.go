package main

import (
	"fmt"
	"os"
	"strings"
	"errors"
	"bytes"
	"sync"
	"time"
	"encoding/xml"
	"github.com/kataras/iris"
	curl "github.com/andelf/go-curl"	
)

var VOTABLESERVER = "jvox.vo.nao.ac.jp"
var VOTABLECACHE = "VOTABLECACHE"

type SubaruDataset struct {	
	dataId string
	processId string
	title string
	date_obs string
	objects string
	band_name string
	band_ref string
	band_hi string
	band_lo string
	band_unit string
	ra string
	dec string
	file_size string
	file_path string
	file_url string
	current_pos int
	data_id_pos int
	process_id_pos int
	title_pos int
	date_obs_pos int
	objects_pos int
	band_name_pos int
	band_ref_pos int
	band_hi_pos int
	band_lo_pos int
	band_unit_pos int
	ra_pos int
	dec_pos int
	file_size_pos int
	file_path_pos int
	file_url_pos int
	timestamp time.Time
	sync.RWMutex
	/*
  sem_t sem_votable ;
  bool has_votable ;
  sem_t sem_fits ;
  bool has_fits ;
  SubaruFITS* fits ;
  sem_t sem_sessions ;*/
}

var datasets = struct{
    sync.RWMutex
    subaru map[string] SubaruDataset
}{subaru: make(map[string] SubaruDataset)}

var easy *curl.CURL

func parseXML(subaru *SubaruDataset, fp *os.File) {
	xml.NewDecoder(fp)
}

func subaru_votable(subaru *SubaruDataset, votable string) {
	//get a votable
	filename := VOTABLECACHE + "/" + subaru.dataId + ".xml"
	xmlfile, err := os.Open(filename)
	defer xmlfile.Close()

	if err != nil {

		tmpfile, err := os.Create(filename+".tmp")
		defer tmpfile.Close()

		if(err != nil) {
			panic(err)
		}

		if len(strings.TrimSpace(votable)) > 0 {
			easy.Setopt(curl.OPT_URL, votable)
		} else {			
			url := "http://" + VOTABLESERVER + ":8060/skynode/do/tap/spcam/sync?REQUEST=queryData&QUERY=SELECT%20*%20FROM%20image_nocut%20WHERE%20data_id%20='" + subaru.dataId + "'"
			fmt.Printf("%s\n",url)
			easy.Setopt(curl.OPT_URL, url)
		}

		// make a callback function
		fooTest := func (buf []byte, userdata interface{}) bool {
			println("DEBUG: size=>", len(buf))
			println("DEBUG: content=>", string(buf))

			file := userdata.(*os.File)
			
			// write a chunk		
			if _, err := file.Write(buf) ; err != nil {
				panic(err)
			}
			
			return true
		}

		easy.Setopt(curl.OPT_WRITEFUNCTION, fooTest)
		easy.Setopt(curl.OPT_WRITEDATA, tmpfile)

		if err := easy.Perform(); err != nil {
			fmt.Printf("ERROR: %v\n", err)
			panic(err)
		} else {
			os.Rename(filename+".tmp", filename)
			xmlfile, err := os.Open(filename)
			defer xmlfile.Close()

			if(err != nil) {
				panic(err)
			}

			parseXML(subaru, xmlfile)
		}
	} else {
		parseXML(subaru, xmlfile)
	}
}

func launch_subaru(dataId, url, votable string) SubaruDataset {
	datasets.RLock()
	subaru, ok := datasets.subaru[dataId]
	datasets.RUnlock()
	
	if(!ok) {
		fmt.Printf("no dataset found, creating a new one\n")

		var subaru SubaruDataset

		subaru.dataId = dataId
		subaru.current_pos = -1
		subaru.data_id_pos = -1
		subaru.process_id_pos = -1
		subaru.title_pos = -1
		subaru.date_obs_pos = -1
		subaru.band_name_pos = -1
		subaru.band_ref_pos = -1
		subaru.band_hi_pos = -1
		subaru.band_lo_pos = -1
		subaru.band_unit_pos = -1
		subaru.ra_pos = -1
		subaru.dec_pos = -1
		subaru.file_size_pos = -1
		subaru.file_path_pos = -1
		subaru.file_url_pos = -1
		subaru.timestamp = time.Now()

		subaru_votable(&subaru, votable)
		
		datasets.Lock()
		datasets.subaru[dataId] = subaru
		datasets.Unlock()

		return subaru
	} else {		
		subaru.Lock()
		subaru.timestamp = time.Now()
		subaru.Unlock()

		return subaru
	}		
}

func execute_subaru(dataId, url, votable string) (bytes.Buffer, error) {
	var buffer bytes.Buffer	
	
	if len(strings.TrimSpace(dataId)) > 0 {
		buffer.WriteString("<h1>")
		buffer.WriteString("<p>URL:&nbsp;")
		buffer.WriteString(url)
		buffer.WriteString("</p>")
		buffer.WriteString("<p>VOTable:&nbsp;")
		buffer.WriteString(votable)
		buffer.WriteString("</p>")
		buffer.WriteString("<p>dataId:&nbsp;")
		buffer.WriteString(dataId)
		buffer.WriteString("</p>")
		buffer.WriteString("</h1>")

		subaru := launch_subaru(dataId, url, votable)

		fmt.Printf("dataId: %s\ttimestamp: %s\n", subaru.dataId, subaru.timestamp.String())
		
		return buffer, nil
	}

	return buffer, errors.New("execute_subaru error")
}

func main() {
	easy = curl.EasyInit()
	defer easy.Cleanup()
	
	fmt.Printf("SubaruWebQL HTTP/WebSocket daemon started.\n")
	
	app := iris.New()	

	app.Get("/subaruwebql/SubaruWebQL.html", func(ctx iris.Context) {
		url := ctx.FormValue("url")
		votable := ctx.FormValue("votable")
		dataId := ctx.FormValue("dataId")		

		page, err := execute_subaru(dataId, url, votable)			

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Writef("SubaruWebQL Internal Server Error\nURL: %s\nVOTable: %s\ndataId: %s", url, votable, dataId)			
		} else {
			ctx.HTML(page.String())
		}
	})

	//root is at http://localhost:8081/subaruwebql/subaru.html
	app.StaticWeb("/", "./htdocs/")	
	app.Favicon("./htdocs/favicon.ico")			
	
	// Start the server using a network address.
	app.Run(iris.Addr(":8081"))

	fmt.Printf("SubaruWebQL HTTP/WebSocket daemon ended.\n")
}
