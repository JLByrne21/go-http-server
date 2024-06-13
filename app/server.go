package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
)

var allowedEncoding = [1]string{"gzip"}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		buffer := make([]byte, 2048)
		n, err := conn.Read(buffer)

		if err != nil {
			return
		}
		recieved := string(buffer[:n])
		basePath := getPath(recieved)
		headerNames := getHeaderNames(recieved)
		content := getContent(recieved)
		method := getMethod(recieved)
		encoding := strings.Split(getHeaderValue(recieved, "accept-encoding"), ",")
		var encodingMethod string
		for _, method := range encoding {
			method = strings.Trim(method, " ")
			if checkEncoding(method) {
				encodingMethod = method
			} 
		}
		noSlashContent := strings.Replace(content, "/", "", -1)
		if basePath == "/" {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else if strings.HasPrefix(basePath, "/echo/") {
			response := responseBuilder(200, "text/plain", encodingMethod, content)
			conn.Write([]byte(response))
		} else if slices.Contains(headerNames, strings.ToLower(noSlashContent)) {
			headerValue := getHeaderValue(recieved, noSlashContent)
			response := responseBuilder(200, "text/plain", encodingMethod, headerValue)
			conn.Write([]byte(response))
		} else if strings.HasPrefix(basePath, "/files/") {
			directory := os.Args[2]
			fileName := strings.TrimPrefix(basePath, "/files/")
			if method == "POST" {

				body := getBody(recieved)


				err = os.WriteFile(directory+fileName, []byte(body), 0644)
				if err != nil {
					response := responseBuilder(500, "text/plain", encodingMethod, "")
					conn.Write([]byte(response))
				} else {
					response := responseBuilder(201, "", encodingMethod, "")
					conn.Write([]byte(response))
				}

			} else {
				data, err := os.ReadFile(directory + fileName)
				if err != nil {
					response := responseBuilder(404, "text/plain", encodingMethod, "")
					conn.Write([]byte(response))
				} else {
					stringData := string(data)
					response := responseBuilder(200, "application/octet-stream", encodingMethod, stringData)
					conn.Write([]byte(response))
				}
			}
		} else {
			conn.Write([]byte(responseBuilder(404, "text/plain", encodingMethod, "")))
		}
	}

}

func checkEncoding(encoding string) bool {
	for _, method := range allowedEncoding {
		if method == encoding {
			return true
		}
	}
	return false
}

func getMethod(request string) string {
	line := strings.Split(request, "\r\n")[0]
	method := strings.Split(line, " ")[0]
	return method
}

func getBody(request string) string {
	line := strings.Split(request, "\r\n")
	return line[4]
}

func getContent(request string) string {
	line := strings.Split(request, "\r\n")[0]
	fullPath := strings.Split(line, " ")[1]
	var content string
	if strings.HasPrefix(fullPath, "/echo/") {
		content = fullPath[len("/echo/"):]
	} else {
		content = fullPath
	}
	return content

}

func getPath(request string) string {
	line := strings.Split(request, "\r\n")[0]
	fullPath := strings.Split(line, " ")[1]
	return fullPath
}

func getHeaderNames(request string) []string {
	lines := strings.Split(request, "\r\n")[1:]
	var headers = []string{}
	for _, h := range lines {
		header_name := strings.Split(h, " ")[0]
		header_name = strings.Replace(header_name, ":", "", -1)
		headers = append(headers, strings.ToLower(header_name))
	}
	return headers
}

func getHeaderValue(request string, headerName string) string {
	lines := strings.Split(request, "\r\n")[1:]
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), headerName) {
			removedName := strings.Split(line, " ")
			if len(removedName) > 1 {
				return strings.Join(removedName[1:], "")
			}
			return removedName[1]
		}
	}
	return ""
}

func responseBuilder(response_code int, content_type string, encoding string, content string) string {
	response := "HTTP/1.1 "
	if response_code == 200 {
		response += "200 OK\r\n"
	} else if response_code == 201 {
		response += "201 Created"
	} else if response_code == 404 {
		response += "404 Not Found"
	} else if response_code == 500 {
		response += "500 Internal Server Error"
	}

	if response_code != 200 {
		response += "\r\n\r\n"
	}
	if encoding != "" {
		response += "Content-Encoding: " + encoding + "\r\n"
	}

	if content_type != "" && content != "" {
		if strings.ToLower(encoding) == "gzip" {
			content = string(gzipEncoder(content))
		}
		contentLength := strconv.Itoa(len(content))
		response += "Content-type: " + content_type + "\r\n"
		response += "Content-length: " + contentLength + "\r\n\r\n"
		response += content + "\r\n\r\n"
	}
	return response
}


func gzipEncoder(body string) []byte {
	var b bytes.Buffer
	gzipWriter := gzip.NewWriter(&b)
	_, err := gzipWriter.Write([]byte(body))
	if err != nil {
		log.Fatal(err)
	}
	gzipWriter.Close()
	return b.Bytes()
	
}