package main

import "fmt"
import "github.com/kataras/iris"

func main() {
	fmt.Printf("SubaruWebQL HTTP/WebSocket daemon\n")

	app := iris.New()

	/*app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<H1>SubaruWebQL HTTP/WebSocket Server</H1>")
	})*/

	app.StaticWeb("/subaruwebql", "./htdocs/subaruwebql")
	app.StaticWeb("/", "./htdocs/subaru.html")
	app.Favicon("./htdocs/favicon.ico")
	
	// Start the server using a network address.
	app.Run(iris.Addr(":8081"))
}
