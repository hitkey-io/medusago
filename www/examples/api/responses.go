package api

import (
	nethttp "net/http"
	"time"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

// region: json
func customGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{
		"message": "Hello, World!",
	})
}

// endregion: json

// region: status
func customPOST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.Status(201).JSON(map[string]any{
		"message": "Created!",
	})
}

// endregion: status

// region: xml
func reportXML(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	res.SetHeader("Content-Type", "application/xml; charset=utf-8")
	res.Send(`<?xml version="1.0" encoding="UTF-8"?><report><status>ok</status></report>`)
	return nil
}

// endregion: xml

// region: file
func invoiceFile(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	// assuming you have the file's content as a byte slice
	content := []byte("%PDF-1.4 ...")

	res.SetHeader("Content-Type", "application/pdf")
	res.SetHeader("Content-Disposition", `attachment; filename="invoice.pdf"`)
	res.Send(string(content))
	return nil
}

// endregion: file

// region: stream
func streamGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	res.SetHeader("Content-Type", "text/event-stream")
	res.SetHeader("Cache-Control", "no-cache")
	res.SetHeader("Connection", "keep-alive")
	res.Status(200)

	// Writer sends the status and headers, then gives access to the
	// underlying writer.
	w := res.Writer()
	flusher, _ := w.(nethttp.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _ = w.Write([]byte("data: Streaming data...\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		case <-req.Request.Context().Done():
			// the client closed the connection
			return nil
		}
	}
}

// endregion: stream

// ResponseRoutes are the route files of the responses chapter.
var ResponseRoutes = []medusahttp.RouteFile{
	{Path: "/custom", Route: medusahttp.Route{GET: customGET, POST: customPOST}},
	{Path: "/custom/report.xml", Route: medusahttp.Route{GET: reportXML}},
	{Path: "/custom/invoice.pdf", Route: medusahttp.Route{GET: invoiceFile}},
	{Path: "/custom/stream", Route: medusahttp.Route{GET: streamGET}},
}
