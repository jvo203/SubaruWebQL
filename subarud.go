package main

import (
	"fmt"
	"math"
	"encoding/binary"
	"os"	
	"io/ioutil"
	"strings"
	"errors"
	"bytes"
	"sync"
	"time"
	"html"
	"strconv"
	"encoding/xml"
	"compress/gzip"
	"github.com/kataras/iris"
	curl "github.com/andelf/go-curl"	
)


var SERVER_STRING = "SubaruWebQL v1.1.0"
var VERSION_STRING = "SV2018-03-14.0"

const NOTIFICATION_CHUNK = 10*1024*1024
const FITS_HEADER_LENGTH = 2880
const FITS_LINE_LENGTH = 80

var VOTABLESERVER = "jvox.vo.nao.ac.jp"
var VOTABLECACHE = "VOTABLECACHE"
var FITSCACHE = "FITSCACHE"

type downloadChunk struct {
	fp *os.File
	previous_size int64
	size int64
	progress int
	subaru *SubaruDataset
	//zlib
	gzip bool
	buf bytes.Buffer
}

const NBINS = 1024

type FITS struct {
	BITPIX int
	NAXIS int
	width int
	height int
	data []float32
	IGNRVAL float32
	CRVAL1 float32
	CDELT1 float32
	CRPIX1 float32
	CRVAL2 float32
	CDELT2 float32
	CRPIX2 float32
	CD1_1 float32
	CD1_2 float32
	CD2_1 float32
	CD2_2 float32
	min float32
	max float32
	hist [NBINS]int
	median float32
	mad float32
	black float32
	sensitivity float32
	rgb []byte
}

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
	file_size int64
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
	fits FITS
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

func round(f float64) float64 {
    return math.Floor(f + .5)
}

type Node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []Node     `xml:",any"`
}

func (n *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node Node

	return d.DecodeElement((*node)(n), &start)
}

func walk(nodes []Node, f func(Node) bool) {
	for _, n := range nodes {
		if f(n) {
			walk(n.Nodes, f)
		}
	}
}

func parseXMLVOTable(subaru *SubaruDataset, fp *os.File) {
	dec := xml.NewDecoder(fp)

	var n Node
	err := dec.Decode(&n)
	if err != nil {
		panic(err)
	}	
	
	walk([]Node{n}, func(n Node) bool {
		if n.XMLName.Local == "FIELD" {
			/*fmt.Println("CONTENT:\t" + string(n.Content))
			fmt.Println(n.Attrs)
			fmt.Println("len(ATTRS) = ", len(n.Attrs))*/
			for i := 0; i < len(n.Attrs); i++ {
				//fmt.Println(n.Attrs[i])

				if(strings.Contains(n.Attrs[i].Name.Local, "ID")) {					
					fmt.Sscanf(n.Attrs[i].Value, "C%d", &subaru.current_pos)
					//fmt.Println("ID:", n.Attrs[i].Value, "current_pos = ", subaru.current_pos)
				}

				if(strings.Contains(n.Attrs[i].Name.Local, "name")) {
					//fmt.Println("name:", n.Attrs[i].Value)

					switch n.Attrs[i].Value {
					case "DATA_ID":
						subaru.data_id_pos = subaru.current_pos
					case "PROC_ID":
						subaru.process_id_pos = subaru.current_pos
					case "TITLE":
						subaru.title_pos = subaru.current_pos
					case "DATE_OBS":
						subaru.date_obs_pos = subaru.current_pos
					case "OBJECTS":
						subaru.objects_pos = subaru.current_pos
					case "BAND_NAME":
						subaru.band_name_pos = subaru.current_pos
					case "BAND_REFVAL":
						subaru.band_ref_pos = subaru.current_pos
					case "BAND_HILIMIT":
						subaru.band_hi_pos = subaru.current_pos
					case "BAND_LOLIMIT":
						subaru.band_lo_pos = subaru.current_pos
					case "BAND_UNIT":
						subaru.band_unit_pos = subaru.current_pos
					case "CENTER_RA":
						subaru.ra_pos = subaru.current_pos
					case "CENTER_DEC":
						subaru.dec_pos = subaru.current_pos
					case "FILE_SIZE":
						subaru.file_size_pos = subaru.current_pos
					case "PATH":
						subaru.file_path_pos = subaru.current_pos
					case "ACCESS_REF":
						subaru.file_url_pos = subaru.current_pos
					default:
					}//end-of-switch
				}
			}
		}//end-of-"FIELD"

		if n.XMLName.Local == "TR" {
			subaru.current_pos = 0
		}//end-of-"TR"

		if n.XMLName.Local == "TD" {
			subaru.current_pos++

			switch subaru.current_pos {			
			case subaru.process_id_pos:
				subaru.processId = string(n.Content)
				
			case subaru.title_pos:
				subaru.title = string(n.Content)

			case subaru.date_obs_pos:
				subaru.date_obs = string(n.Content)

			case subaru.objects_pos:
				subaru.objects = string(n.Content)

			case subaru.band_name_pos:
				subaru.band_name = string(n.Content)

			case subaru.band_ref_pos:
				subaru.band_ref = string(n.Content)

			case subaru.band_hi_pos:
				subaru.band_hi = string(n.Content)

			case subaru.band_lo_pos:
				subaru.band_lo = string(n.Content)

			case subaru.band_unit_pos:
				subaru.band_unit = string(n.Content)

				if(subaru.band_unit == "A") {
					subaru.band_unit = "&#8491;"
				}
				
				if(subaru.band_unit == "um") {
					subaru.band_unit = "&#181;m"
				}			

			case subaru.ra_pos:
				subaru.ra = string(n.Content)

			case subaru.dec_pos:
				subaru.dec = string(n.Content)

			case subaru.file_size_pos:
				i64, err := strconv.ParseInt(string(n.Content), 10, 64)

				if(err == nil) {
					subaru.file_size = i64
				}

			case subaru.file_path_pos:
				subaru.file_path = string(n.Content)

			case subaru.file_url_pos:
				subaru.file_url = html.UnescapeString(string(n.Content))

			default:
			}
		}//end-of-"TD"
		
		return true
	})

	fmt.Printf("%+v\n", subaru)
}

func subaru_votable(subaru *SubaruDataset, votable string) {	
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
			//fmt.Printf("%s\n",url)
			easy.Setopt(curl.OPT_URL, url)
		}

		// make a callback function
		writeFile := func (buf []byte, userdata interface{}) bool {
			/*println("DEBUG: size=>", len(buf))
			println("DEBUG: content=>", string(buf))*/

			file := userdata.(*os.File)
			
			// write a chunk		
			if _, err := file.Write(buf) ; err != nil {
				panic(err)
			}
			
			return true
		}

		easy.Setopt(curl.OPT_WRITEFUNCTION, writeFile)
		easy.Setopt(curl.OPT_WRITEDATA, tmpfile)

		if err := easy.Perform(); err != nil {
			fmt.Printf("ERROR: %+v\n", err)
			panic(err)
		} else {
			os.Rename(filename+".tmp", filename)
			
			xmlfile, err := os.Open(filename)
			defer xmlfile.Close()

			if(err != nil) {
				panic(err)
			}

			parseXMLVOTable(subaru, xmlfile)
		}
	} else {
		parseXMLVOTable(subaru, xmlfile)
	}
}

func read_FITS_bytes(buf []byte, offset int, dest []float32) {

	fmt.Println("len(slice):", len(buf))
	
	pos := 0
	n := len(buf) / 4
	
	for i:= 0; i < n; i++ {
		floatBuf := buf[pos:pos+4]
		bits := binary.BigEndian.Uint32(floatBuf)
		float := math.Float32frombits(bits)
		dest[offset+i] = float
		pos += 4

		if(i > n - 4) {
			fmt.Println("bytes:", floatBuf, "float:", dest[offset+i])
		}
	}
}

func read_FITS_from_buffer(subaru *SubaruDataset, buffer *bytes.Buffer) {
	fmt.Println("FITS buffer length:", buffer.Len())

	//read the header first
	hdrLine := make([]byte, FITS_LINE_LENGTH)
	total := 0
	fitsLen := buffer.Len()
	hend := false	

	for total < fitsLen && !hend {
		count, _ := buffer.Read(hdrLine)
		total += count

		//fmt.Println("read", count, "bytes")
		s := string(hdrLine)		
		//fmt.Println(string(s))

		if(strings.Contains(s, "END       ")) {
			hend = true
		}

		if(strings.Contains(s, "BITPIX  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%d", &subaru.fits.BITPIX)
		}

		if(strings.Contains(s, "NAXIS   = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%d", &subaru.fits.NAXIS)
		}

		if(strings.Contains(s, "NAXIS1  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%d", &subaru.fits.width)
		}

		if(strings.Contains(s, "NAXIS2  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%d", &subaru.fits.height)
		}

		if(strings.Contains(s, "IGNRVAL = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.IGNRVAL)
		}

		if(strings.Contains(s, "CRVAL1  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CRVAL1)
		}

		if(strings.Contains(s, "CDELT1  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CDELT1)
		}

		if(strings.Contains(s, "CRPIX1  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CRPIX1)
		}

		if(strings.Contains(s, "CRVAL2  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CRVAL2)
		}

		if(strings.Contains(s, "CDELT2  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CDELT2)
		}

		if(strings.Contains(s, "CRPIX2  = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CRPIX2)
		}

		if(strings.Contains(s, "CD1_1   = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CD1_1)
		}

		if(strings.Contains(s, "CD1_2   = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CD1_2)
		}

		if(strings.Contains(s, "CD2_1   = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CD2_1)			
		}

		if(strings.Contains(s, "CD2_2   = ")) {
			fmt.Println(s[10:])
			fmt.Sscanf(s[10:], "%f", &subaru.fits.CD2_2)
		}
	}

	rem := total % FITS_HEADER_LENGTH	
	offset := (total - rem)
	
	if(rem > 0) {
		offset += FITS_HEADER_LENGTH
		rem = offset - total

		dummy := make([]byte, rem)
		_, _ = buffer.Read(dummy)
	}	

	fmt.Println("FITS HEADER LENGTH:", offset, "total:", total, "rem:", rem)
	fmt.Println("data size:", subaru.fits.width*subaru.fits.height*4)
	fmt.Printf("%+v\n", subaru.fits)

	if(subaru.fits.BITPIX != -32) {
		panic(errors.New("UNSUPPORTED BITPIX"))
	}

	//FITS DATA BEGINS AT buffer.Bytes()[offset:]
	//need to convert from BIG-ENDIAN to LITTLE-ENDIAN and []byte to float32
	src := buffer.Bytes()
	fmt.Println("src buffer length:", len(src))	

	//first read the remaining part of the FITS HEADER in one go
	//then buffer.Bytes() will point to the remaining unread data (FITS DATA + padding)
	//I can see how/why Go is not efficient in computing applications
	//it might be better to stick with C/C++ or use Rust
	//But after trying Rust a bit, Rust seems too strict, too convoluted
	
	total_size := subaru.fits.width*subaru.fits.height
	subaru.fits.data = make([]float32, total_size)
	
	/*
floatBuf := make([]byte, 4)//four bytes in float32
for i := 0; i < total_size; i++ {
		_, _ = buffer.Read(floatBuf)
		bits := binary.BigEndian.Uint32(floatBuf)
		float := math.Float32frombits(bits)
		subaru.fits.data[i] = float

		if(i > total_size - 10) {
			fmt.Println("bytes:", floatBuf, "float:", float)
		}
	}*/

	//take slices of src
	//process them in parallel
	//read_FITS_bytes(src, 0, subaru.fits.data)

	go read_FITS_bytes(src[0:4*total_size/2], 0, subaru.fits.data)
	go read_FITS_bytes(src[4*total_size/2:4*total_size], total_size/2, subaru.fits.data)	
	
	fmt.Println("subaru_fits_thread finished.")
}

func read_FITS_from_file(subaru *SubaruDataset, fp *os.File) {
	fmt.Println("reading FITS file for", subaru.dataId)

	// get the file size
	stat, err := fp.Stat()
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(fp)

	if(err != nil) {
		panic(err)
	} else {
		fmt.Println("uncompressed size:", len(buf), "file size:", stat.Size())

		read_FITS_from_buffer(subaru, bytes.NewBuffer(buf))
	}
}

func subaru_fits_thread(subaru *SubaruDataset) {	
	filename := FITSCACHE + "/" + subaru.dataId + ".fits"
	
	fitsfile, err := os.Open(filename)
	defer fitsfile.Close()

	if err != nil {
		tmpfile, err := os.Create(filename+".tmp")
		defer tmpfile.Close()

		if(err != nil) {
			panic(err)
		}

		fmt.Printf("subaru_fits_thread: %s\n", subaru.file_url)
		easy.Setopt(curl.OPT_URL, subaru.file_url)

		// make callback functions		
		header_callback := func (buf []byte, userdata interface{}) bool {
			/*println("HEADER: size=>", len(buf))*/
			/*println("HEADER: content=>", string(buf))*/

			header := string(buf)
			chunk := userdata.(*downloadChunk)			

			if(strings.Contains(header, "gzip") || strings.Contains(header, ".fits.gz")) {
				chunk.gzip = true
			}			
			
			return true
		}
		
		writeFile := func (buf []byte, userdata interface{}) bool {
			//println("DEBUG: size=>", len(buf))
			/*println("DEBUG: content=>", string(buf))*/

			chunk := userdata.(*downloadChunk)						
			//subaru := chunk.subaru			
			file := chunk.fp			

			//append buf to chunk.buf
			chunk.buf.Write(buf)
			
			if(!chunk.gzip) {												
				// write a chunk		
				if _, err := file.Write(buf) ; err != nil {
					panic(err)
				}
			}
			
			chunk.size += int64(len(buf))
			chunk.progress = int(round(100.0 * float64(chunk.size) / float64(subaru.file_size)))
			//fmt.Printf("%.0f%%\n",round(100.0 * float64(chunk.size) / float64(subaru.file_size)))

			if( (chunk.size - chunk.previous_size) >= int64(NOTIFICATION_CHUNK)) {
				chunk.previous_size = chunk.size
				//send_progress_notification(subaru.dataId, chunk.size, chunk.progress)
			}
			
			return true
		}		

		chunk := downloadChunk{fp: tmpfile, previous_size: 0, size: 0, subaru: subaru, gzip: false}
		
		easy.Setopt(curl.OPT_WRITEFUNCTION, writeFile)
		easy.Setopt(curl.OPT_WRITEDATA, &chunk)

		easy.Setopt(curl.OPT_HEADERFUNCTION, header_callback)
		easy.Setopt(curl.OPT_HEADERDATA, &chunk)		

		if err := easy.Perform(); err != nil {
			fmt.Printf("ERROR: %+v\n", err)
			panic(err)
		} else {
			fmt.Println("len(chunk.buf):", chunk.buf.Len())

			if(int64(chunk.buf.Len()) != subaru.file_size) {
				panic(errors.New("received wrong amount of data"))
			}
			
			if(chunk.gzip) {				
				gr, err := gzip.NewReader(&chunk.buf)
				defer gr.Close()

				if(err != nil) {
					panic(err)
				} else {
					//read in all uncompressed data
					buf, err := ioutil.ReadAll(gr)

					if(err != nil) {
						panic(err)
					} else {
						fmt.Println("uncompressed size:", len(buf))

						go read_FITS_from_buffer(subaru, bytes.NewBuffer(buf))

						// write a chunk to disk	
						if _, err := tmpfile.Write(buf) ; err != nil {
							panic(err)
						}
					}										
				}
			} else {								
				go read_FITS_from_buffer(subaru, &chunk.buf)
			}
			
			os.Rename(filename+".tmp", filename)						
		}
	} else {
		read_FITS_from_file(subaru, fitsfile)
	}
}

func launch_subaru(dataId, votable string) SubaruDataset {
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

		go subaru_fits_thread(&subaru)

		return subaru
	} else {		
		subaru.Lock()
		subaru.timestamp = time.Now()
		subaru.Unlock()

		return subaru
	}		
}

func execute_subaru(dataId, votable string) (strings.Builder, error) {
	//var buffer bytes.Buffer
	var buffer strings.Builder
	
	if len(strings.TrimSpace(dataId)) > 0 {
		/*buffer.WriteString("<h1>")		
		buffer.WriteString("<p>VOTable:&nbsp;")
		buffer.WriteString(votable)
		buffer.WriteString("</p>")
		buffer.WriteString("<p>dataId:&nbsp;")
		buffer.WriteString(dataId)
		buffer.WriteString("</p>")
		buffer.WriteString("</h1>")*/

		subaru := launch_subaru(dataId, votable)
		fmt.Printf("dataId: %s\ttimestamp: %s\n", subaru.dataId, subaru.timestamp.String())

		buffer.WriteString("<!DOCTYPE html>\n<html xmlns:xlink=\"http://www.w3.org/1999/xlink\">\n<head>\n<meta charset=\"utf-8\">\n")
		buffer.WriteString("<link rel=\"stylesheet\" type=\"text/css\" href=\"http://fonts.googleapis.com/css?family=Inconsolata\">\n")//Orbitron
		buffer.WriteString("<script src=\"https://d3js.org/d3.v4.min.js\"></script>\n")
		buffer.WriteString("<script src=\"/subaruwebql/progressbar.min.js\"></script>\n")
		buffer.WriteString("<script src=\"/subaruwebql/ra_dec_conversion.js\"></script>\n")
		buffer.WriteString("<script src=\"/subaruwebql/reconnecting-websocket.min.js\"></script>\n")
		buffer.WriteString("<script src=\"/subaruwebql/subaruwebql.js\"></script>\n")
		//buffer.WriteString(fmt.Sprintf("<script src=\"/subaruwebql/subaruwebql.js?%s\"></script>\n", VERSION_STRING))
		
		buffer.WriteString("<!-- Latest compiled and minified CSS --> <link rel=\"stylesheet\" href=\"https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css\" integrity=\"sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u\" crossorigin=\"anonymous\">\n")
		buffer.WriteString("<!-- Optional theme --> <link rel=\"stylesheet\" href=\"https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css\" integrity=\"sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp\" crossorigin=\"anonymous\">\n")
		buffer.WriteString("<!-- jQuery (necessary for Bootstrap's JavaScript plugins) --> <script src=\"https://ajax.googleapis.com/ajax/libs/jquery/1.12.4/jquery.min.js\"></script>\n")
		buffer.WriteString("<!-- Latest compiled and minified JavaScript --> <script src=\"https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js\" integrity=\"sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa\" crossorigin=\"anonymous\"></script>\n")

		buffer.WriteString("<link rel=\"stylesheet\" href=\"/subaruwebql/subaruwebql.css\"/>\n")
		buffer.WriteString("<script src=\"/subaruwebql/lz4.min.js\" charset=\"utf-8\"></script>\n")

		buffer.WriteString("<title>SubaruWebQL</title></head><body>\n")
		
		buffer.WriteString(fmt.Sprintf("<div id='votable' style='width: 0; height: 0;' data-dataId='%s' data-processId='%s' data-title='%s' data-date='%s' data-objects='%s' data-band-name='%s' data-band-ref='%s' data-band-hi='%s' data-band-lo='%s' data-band-unit='%s' data-ra='%s' data-dec='%s' data-filesize='%s' data-server-version='%s'></div>\n", dataId, subaru.processId, subaru.title, subaru.date_obs, subaru.objects, subaru.band_name, subaru.band_ref, subaru.band_hi, subaru.band_lo, subaru.band_unit, subaru.ra, subaru.dec, subaru.file_size, VERSION_STRING))

		buffer.WriteString(`<script>
const golden_ratio = 1.6180339887;
var firstTime = true ;
mainRenderer();
window.onresize = resize;
function resize(){mainRenderer();}
  </script>`) ;
		
		buffer.WriteString("</body></html>") ;
		
		return buffer, nil
	}

	return buffer, errors.New("execute_subaru error")
}

func main() {
	easy = curl.EasyInit()
	defer easy.Cleanup()	
	
	app := iris.New()	

	app.Get("/subaruwebql/SubaruWebQL.html", func(ctx iris.Context) {		
		votable := ctx.FormValue("votable")
		dataId := ctx.FormValue("dataId")		

		page, err := execute_subaru(dataId, votable)			

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Writef("SubaruWebQL Internal Server Error\nVOTable: %s\ndataId: %s", votable, dataId)			
		} else {
			ctx.HTML(page.String())
		}
	})

	//root is at http://localhost:8081/subaruwebql/subaru.html
	app.StaticWeb("/", "./htdocs/")	
	app.Favicon("./htdocs/favicon.ico")			
		
	fmt.Printf("%s started.\n", SERVER_STRING)
	
	// Start the server using a network address.
	app.Run(iris.Addr(":8081"))

	fmt.Printf("%s daemon ended.\n", SERVER_STRING)
}
