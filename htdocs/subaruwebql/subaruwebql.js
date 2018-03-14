function get_js_version()
{
    return "JS2018-03-14.1" ;
}

function get_maximum_image_width(width)
{
    //return Math.floor(width) ;
    
    //return 512 ;
    
    //restrict width on high resolution displays (>4K) in order to conserve network bandwith
    return Math.min(1920, Math.floor(width)) ;//stop at Full HD (1920)?
}


function get_screen_scale(x)
{
    return x ;//Math.floor(0.9*x) ;
}

function get_image_scale(width, height, img_width, img_height)
{
    /*if (height < width)
	return get_screen_scale(height)/img_height ;
    else*/
	return get_screen_scale(width)/img_width ;
}


d3.selection.prototype.moveToFront = function() {
    return this.each(function(){
	this.parentNode.appendChild(this);
    });
};

d3.selection.prototype.moveToBack = function() { 
    return this.each(function() { 
	var firstChild = this.parentNode.firstChild; 
	if (firstChild) { 
	    this.parentNode.insertBefore(this, firstChild); 
	} 
    }); 
};

function open_progress_websocket_connection(dataId)
{
    bar = null;
    
    if ("WebSocket" in window)
	{
	    //alert("WebSocket is supported by your Browser!");
               
	    // Let us open a web socket
	    var loc = window.location, ws_uri;
	    if (loc.protocol === "https:")
		{
		    ws_uri = "wss:";
		}
	    else
		{
		    ws_uri = "ws:";
		    //ws_uri = "wss:";//JVO proxy needs wss
		}
	    ws_uri += "//" + loc.host + "/subaruwebql/websocket/progress/" + dataId;
	    //ws_uri += loc.pathname + "/progress/";

	    console.log("ws_uri: " + ws_uri) ;
	    
	    //var ws = new WebSocket(ws_uri, "progress-protocol");
	    try
	    {
		progressWS = new WebSocket(ws_uri);
	    }
	    catch (e)
	    {
		console.log(e) ;
	    } ;

	    progressWS.onerror = function()
	    {
		d3.select("#container").remove();
	    } ;
	    
	    progressWS.onopen = function()
	    {
		// Web Socket is connected, send data using send()
		document.body.style.cursor = "wait";
		//document.body.className = 'wait' ;
		//console.log(document.body.className) ;
		
		//create a progress bar container
		bar = new ProgressBar.Circle('#container', {
		    color: '#aaa',
		    // This has to be the same size as the maximum width to
		    // prevent clipping
		    strokeWidth: 6,
		    trailWidth: 1,
		    color: '#fff',/*'#f1f1f1',/*'#FFEA82',*/
		    trailColor: '#777',
		    svgStyle: {
			display: 'block',

			// Important: make sure that your container has same
			// aspect ratio as the SVG canvas. See SVG canvas sizes above.
			width: '100%',
			height: '100%'
		    },
		    easing: 'linear',//'easeInOut',
		    duration: 250,
		    text: {
			autoStyleContainer: false,
			style: {
			    // Text color.
			    // Default: same as stroke color (options.color)
			    //color: '#f00',
			    position: 'absolute',
			    left: '50%',
			    top: '50%',
			    padding: 0,
			    margin: 0,
			    // You can specify styles which will be browser prefixed
			    transform: {
				prefix: true,
				value: 'translate(-50%, -50%)'
			    }
			},
		    },
		    from: { color: '#777', width: 1 },
		    to: { color: '#fff', width: 6 },
		    // Set default step function for all animate calls
		    step: function(state, circle) {
			circle.path.setAttribute('stroke', state.color);
			circle.path.setAttribute('stroke-width', state.width);

			var value = Math.round(circle.value() * 100);
			if (value === 0) {
			    circle.setText(value + '');
			} else {
			    circle.setText(value + '');
			}

		    }
		});
		//bar.text.style.fontFamily = '"Raleway", Helvetica, sans-serif';
		bar.text.style.fontSize = '5rem';
		
		//progressWS.send(dataId);
		//console.log("Progress WebSocket open request sent.");
	    };

	    progressWS.onmessage = function (evt) 
	    { 
		/*var received_msg = evt.data;
		console.log("Received: " + received_msg) ;*/
		process_progress_event(evt) ;
	    };
				
	    progressWS.onclose = function()
	    { 
		// websocket is closed.
		console.log("Progress WebSocket Connection is closed...");

		document.body.style.cursor = "default";
		
		if(bar != null)
		{
		    bar.destroy() ;

		    d3.select("#container").remove();
		}
		
		if(!has_image)
	    	    d3.select("#jvoText").text("LOADING IMAGE...") ;
	    };
	}
    else
    {
	d3.select("#jvoText").text("LOADING IMAGE...") ;
	
	// The browser doesn't support WebSocket
	alert("WebSocket NOT supported by your Browser, progress updates disabled.");
    }
}

function display_image_info(votable, width, svg)
{
    var ra = ParseRA('+'+votable.getAttribute('data-ra')) ;
    var dec = ParseDec(votable.getAttribute('data-dec')) ;
    var xradec = new Array ( (ra/3600.0) / toDegrees, (dec/3600.0) / toDegrees );
    var offset = 1 ;

    if(votable.getAttribute('data-title').trim() != "")
    {
	svg.append("text")
	    .attr("x", (0.995*width-1))//0
	    .attr("y", offset+"em")
	    .attr("font-family", "Inconsolata")//was Arial, Monospace
	    .attr("font-size", "1.75em")
	    .attr("text-anchor", "end")
	    .attr("stroke", "none")
	    .attr("opacity", 0.9)
	    .text(votable.getAttribute('data-title').replace(/_/g," "));
	
	offset += 1.5;
    }

    if(votable.getAttribute('data-objects').trim() != "")
    {
	svg.append("text")
	    .attr("x", (0.995*width-1))//0
	    .attr("y", offset+"em")
	    .attr("font-family", "Inconsolata")
	    .attr("font-size", "1.5em")
	    .attr("text-anchor", "end")
	    .attr("stroke", "none")
	    .attr("opacity", 0.9)
	    .text(votable.getAttribute('data-objects').replace(/_/g," "));

	offset += 1.25;
    }

    if(votable.getAttribute('data-date').trim() != "")
    {
	svg.append("text")
	    .attr("x", (0.995*width-1))//0
	    .attr("y", offset+"em")
	    .attr("font-family", "Inconsolata")
	    .attr("font-size", "1.5em")
	    .attr("text-anchor", "end")
	    .attr("stroke", "none")
	    .attr("opacity", 0.9)
	    .text(votable.getAttribute('data-date').replace(/_/g," "));

	offset += 1.25;
    }

    if(votable.getAttribute('data-band-ref').trim() != "")
    {
	svg.append("text")
	    .attr("x", (0.995*width-1))//0
	    .attr("y", offset+"em")
	    .attr("font-family", "Inconsolata")
	    .attr("font-size", "1.5em")
	    .attr("text-anchor", "end")
	    .attr("stroke", "none")
	    .attr("opacity", 0.9)
	    .text(/*votable.getAttribute('data-band-name') + ' ' + */votable.getAttribute('data-band-ref').replace(/_/g," ") + votable.getAttribute('data-band-unit'));

	offset += 1.25;
    }

    svg.append("text")
	.attr("id", "ra")
	.attr("x", (0.995*width-1))
	.attr("y", offset+"em")
	.attr("font-family", "Inconsolata")
	.attr("font-size", "1.5em")
	.attr("text-anchor", "end")
	.attr("stroke", "none")
	.text(RadiansPrintHMS(xradec[0])) ;
    
    offset += 1;
    
    svg.append("text")
	.attr("id", "dec")
	.attr("x", (0.995*width-1))
	.attr("y", offset+"em")
	.attr("font-family", "Inconsolata")
	.attr("font-size", "1.5em")
	.attr("text-anchor", "end")
	.attr("stroke", "none")
	.text(RadiansPrintDMS(xradec[1])) ;
}

function display_image()
{
    if(firstTime)
    {
	has_image = false ;
	
	//preload an image
	var img = new Image();
	img.onload = function () {

	    var new_img = document.getElementById("SubaruImageContainer").firstChild ;
	    
	    if(new_img == null)
	    {
		var c = document.getElementById("BackHTMLCanvas");
		var width = c.width;
		var height = c.height;
		var ctx = c.getContext("2d");

		/*ctx.mozImageSmoothingEnabled = false;
		  ctx.webkitImageSmoothingEnabled = false;
		  ctx.msImageSmoothingEnabled = false;
		  ctx.imageSmoothingEnabled = false;*/

		//ctx.webkitFilter = "invert(100%)";
		//ctx.filter = "invert(100%)";
		
		if(img != null)
		{
		    var scale = get_image_scale(width, height, img.width, img.height) ;
		    var img_width = scale*img.width ;
		    var img_height = scale*img.height;
		    //ctx.drawImage(img, (width-img_width)/2, (height-img_height)/2, img_width, img_height);
		    ctx.drawImage(img, 0, 0, img_width, img_height);
		} ;
	    } ;

	    //needed to put this line in the last position due to a bug in IE11
	    document.getElementById("JVOImageContainer").appendChild(this);
	}
	var image_url = "http://jvo.nao.ac.jp/portal/subaru/spcam.do?procId=" + votable.getAttribute('data-processid') + "&action=mosaicThumbnail&mosaicId=" + votable.getAttribute('data-dataid') ;
	img.src= image_url ;
    }
}

function mainRenderer()
{
    console.log("mainRenderer");
        
    $("body").toggleClass("wait");
    
    var votable = document.getElementById("votable");

    if(firstTime)
    {
	d3.select("body").append("div")
	    .attr("id", "JVOImageContainer")
	    .style("display", "none") ;

	d3.select("body").append("div")
	    .attr("id", "SubaruImageContainer")
	    .style("display", "none") ;	
    }//end-of if(firstTime)
    else//if not firstTime
	d3.select("#mainCanvas").remove();

    var div = d3.select("body").append("div")
	.attr("id", "mainCanvas")
	.attr("class", "canvasDiv") ;

    var rect = document.getElementById('mainCanvas').getBoundingClientRect();
    var width = rect.width ;//- 20 ;
    var height = rect.height ;//- 20 ;

    //set the default font-size (1em)
    emFontSize = Math.max(12, 0.011 * 0.5*(width + height)) ;
    emStrokeWidth = 0.1*emFontSize ;
    document.body.style.fontSize = emFontSize + "px" ;
    console.log("emFontSize : ", emFontSize, "emStrokeWidth : ", emStrokeWidth) ;

    d3.select("#mainCanvas").append("canvas")
	.attr("id", "BackHTMLCanvas")
	.attr("width", width)
	.attr("height", height)
	.attr('style', 'position: fixed; left: 0px; top: 0px; z-index: 0');
    
    d3.select("#mainCanvas").append("canvas")
	.attr("id", "HTMLCanvas")
	.attr("width", width)
	.attr("height", height)
	.attr('style', 'position: fixed; left: 0px; top: 0px; z-index: 1');

    d3.select("#mainCanvas").append("svg")
	.attr("id", "SVGGridCanvas")
	.attr("width", width)
	.attr("height", height)
	.attr('style','position: fixed; left: 0px; top: 0px; z-index: 2; cursor: inherit; mix-blend-mode: none');
    
    d3.select("#mainCanvas").append("canvas")
	.attr("id", "ZOOMCanvas")
	.attr("width", width)
	.attr("height", height)
	.attr('style', 'position: fixed; left: 0px; top: 0px; z-index: 3');

    var svg = d3.select("#mainCanvas").append("svg")
	.attr("id", "SVGCanvas")
	.attr("width", width)
	.attr("height", height)
	.attr('style','position: fixed; left: 0px; top: 0px; z-index: 5; cursor: inherit; mix-blend-mode: none');

    svg.append("text")
	.attr("id", "jvoText")
	.attr("x", width/2)
	.attr("y", height/2)
	.attr("font-family", "Inconsolata")
	.attr("font-weight", "bold")
	.attr("font-size", "3em")
	.attr("text-anchor", "middle")
	.attr("fill", "white")
	.attr("stroke", "black")
	.attr("pointer-events", "none")
	.attr("opacity", 1.0)
	.text("") ;

    svg.append("svg:image")
	.attr("id", "jvoLogo")
	.attr("x", (width-1-199))
	.attr("y", (height-1-109))
	.attr("xlink:href", "http://jvo.nao.ac.jp/images/JVO_logo_199x109.png")	
	.attr("width", 199)
	.attr("height", 109)
	.attr("opacity", 0.5) ;

    display_image_info(votable, width, svg) ;
    display_image() ;

    if(firstTime)
    {
	d3.select("#mainCanvas").append("div")
	    .attr("id", "container");
	
	open_progress_websocket_connection(votable.getAttribute('data-dataid'));
    }

    //if(firstTime)
    //open_image_websocket_connection(votable.getAttribute('data-dataid'));
    
    firstTime = false ;
    $("body").toggleClass("wait");    
}
