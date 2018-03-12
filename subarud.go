package main

import "fmt"
import "bytes"
import "github.com/kataras/iris"

func main() {
	fmt.Printf("SubaruWebQL HTTP/WebSocket daemon started.\n")

	app := iris.New()	
	
	app.StaticWeb("/subaruwebql", "./htdocs/subaruwebql")	
	app.Favicon("./htdocs/favicon.ico")	

	app.Get("/subaruwebql/SubaruWebQL.html", func(ctx iris.Context) {
		url := ctx.FormValue("url")
		votable := ctx.FormValue("votable")
		dataId := ctx.FormValue("dataId")

		var buffer bytes.Buffer
		
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

		//page, error = execute_subaru(dataId, votable, url) ;
		
		//ctx.Writef("URL: %s, VOTable: %s, dataId: %s", url, votable, dataId)
		ctx.HTML(buffer.String())
	})

	//root is at http://localhost:8081/subaruwebql/subaru.html
	
	// Start the server using a network address.
	app.Run(iris.Addr(":8081"))

	fmt.Printf("SubaruWebQL HTTP/WebSocket daemon ended.\n")
}
