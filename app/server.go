package main

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
)

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
		noSlashContent := strings.Replace(content, "/", "", -1)
		if basePath == "/" {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else if strings.HasPrefix(basePath, "/echo/") {
			response := responseBuilder(200, "text/plain", content)
			conn.Write([]byte(response))
		} else if slices.Contains(headerNames, strings.ToLower(noSlashContent)) {
			headerValue := getHeaderValue(recieved, noSlashContent)
			fmt.Print(headerValue)
			response := responseBuilder(200, "text/plain", headerValue)
			conn.Write([]byte(response))
		} else if strings.HasPrefix(basePath, "/files/") {
			directory := os.Args[2]
			fileName := strings.TrimPrefix(basePath, "/files/")
			data, err := os.ReadFile(directory + fileName)
			if err != nil {
				response := responseBuilder(404, "text/plain", "")
				conn.Write([]byte(response))
			} else {
				stringData := string(data)
				response := responseBuilder(200, "application/octet-stream", stringData)
				conn.Write([]byte(response))
			}

		} else {
			conn.Write([]byte(responseBuilder(404, "text/plain", "")))
		}
	}

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
			return strings.Split(line, " ")[1]
		}
	}
	return ""
}

func responseBuilder(response_code int, content_type string, content string) string {
	response := ""
	if response_code == 200 {
		response += "HTTP/1.1 200 OK\r\n"
	} else if response_code == 404 {
		response += "HTTP/1.1 404 Not Found\r\n\r\n"
	}

	if content_type != "" && content != "" {
		contentLength := strconv.Itoa(len(content))
		response += "Content-type: " + content_type + "\r\n"
		response += "Content-length: " + contentLength + "\r\n\r\n"
		response += content + "\r\n\r\n"
	}
	return response
}
