package main

import (
	"fmt"
	"os"
	"net"
	"time"
	"sync"
	"io/ioutil"
	"strings"
	"strconv"
	"os/signal"
	"os/exec"
	"syscall"
	"net/http"
)

type RequestCache struct {
	Response	http.ResponseWriter
	Request		*http.Request
}

var (
	mutex sync.Mutex
	cmd *exec.Cmd
	sig os.Signal
	cpu_percent = make(chan int)
	index int	

	// parameters for evaluation
	parse_req = make([]int, 0)
	self_req = make([]int, 0)
	neighbor_req = make([]int, 0)
	index_parse int 
	index_add int
	index_remove int

	queue []RequestCache
)

const (
	SECOND time.Duration = 1
	MILLI_main time.Duration = 100
	MILLI_sub time.Duration = 500

	tcp_port string = ":8002"
	udp_port string = ":8001"
	udp_port_serv string = ":8001"
)

// copy and paste
// time.Sleep(SECOND * time.Second)
// time.Sleep(MILLI_main * time.Millisecond)
// time.Sleep(MILLI_sub * time.Millisecond)

func init() {
	cmd = exec.Command("sh", "cpu.sh")
	fmt.Println("123")
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	fmt.Println("get")
}

func main() {
	// variant setting
	c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	fmt.Printf("%T \n", c)

	// 4 process running

	go ReadCPU_usage(/* cpu_percent */)

	go HTTPServer()

	go Socket_Server()

	for {
		select {
		case sig = <-c:
			fmt.Println("process finish")
			final()
			cmd.Process.Kill()
			return
		default:
			cpu := <-cpu_percent
			fmt.Println("current:", cpu)
		
			time.Sleep(SECOND * time.Second)
		}
	}
}

func final() {
	fmt.Println("final")
	fmt.Printf("requests: all %d, current %d, response %d parse %d\n", index_add, len(queue), index_remove, index_parse)
    fmt.Printf("parse: %d\n", parse_req)
}

func HTTPServer() {
	// http handle func
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("---")
		index++

		requestCache := RequestCache {
			Response:	w,
			Request:	r,
		}
		//fmt.Println("body: ", requestCache.Index, requestCache.Response, requestCache.Request)
		
		queue = append(queue, requestCache)
		
		//fmt.Println("outline: ", queue)
		fmt.Println("current: ", index)
		// add proxy
		if index > 10 {
			//go Socket_Client()
		}

	})
	
	// HTTP Server
	fmt.Printf("HTTP server is listening on %s...\n", tcp_port)
	http.ListenAndServe(tcp_port, nil)

}

func Socket_Server() {
	// waiting connection from src container, send feedback to src container
	// set own [:Port]
	//service := ":8010"

	// resolve UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", udp_port_serv)
  	if err != nil {
		fmt.Println("Error resolving address:", err)
    	os.Exit(1)
  	}

	// bind after creating a socket
	conn, err := net.ListenUDP("udp", udpAddr)
  	if err != nil {
		fmt.Println("Error opening UDP connection:", err)
    	os.Exit(1)
  	}
  	defer conn.Close()

  	fmt.Println("Server is listening on", udp_port_serv)

	buffer := make([]byte, 1024)
	// if a connection is established, send data to client
	for {
		// wait for data to come from client
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Unable to receive data", err)
			continue
		}
		str := string(buffer[:n])
		fmt.Println("<<<: ", str, n) // number of byte: 6

		// send own number of requests to client
		
		/*
		// case CPU Usage:
		cpu := <- cpu_percent

		// convert Int to String
		cpu_str := strconv.Itoa(cpu)

		buf := []byte(cpu_str)
		*/
		//_, err = conn.Write([]byte(fmt.Sprintf("%d", index)))

		// case number of request
		req_str := strconv.Itoa(index)
		buf := []byte(req_str)

		fmt.Println(buf)

		_, err = conn.WriteToUDP(buf, addr)
		if err != nil {
			fmt.Println("Unable to send data", err)
			continue
		}

		//fmt.Println("sent data: ", cpu)
		fmt.Println("sent data: ", req_str)
	}
}

/*
func Socket_Client() int {
	// connect to neighbor container
	// set server [IP address:Port] to connect
	serverAddr := "172.17.0.4" + udp_port

	// resolve server UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
  	if err != nil {
		fmt.Println("Error resolving address:", err)
    	os.Exit(1)
  	}

	// bind after make socket
	conn, err := net.DialUDP("udp", nil, udpAddr)
  	if err != nil {
		fmt.Println("Error opening UDP connection:", err)
    	os.Exit(1)
  	}
  	defer conn.Close()

	// if a connection is established, receive data from server
	// send something data to server
	_, err = conn.Write([]byte("please"))
	if err != nil {
		fmt.Println("Unable to send message", err)
	}
	
	// receive CPU Usage data (actually number of requests)
	data := make([]byte, 1024)
	req_data, err := conn.Read(data)
	if err != nil {
		fmt.Println("No data:", err)
    	os.Exit(1)
	}

	req_trim := strings.TrimSpace(string(req_data))
	req, err := strconv.Atoi(req_trim)
	if err != nil {
		fmt.Println("failure get request", err)

	}

	fmt.Println(req)

	return req
}
*/

func ReadCPU_usage() {
	ticker := time.NewTicker(SECOND * time.Second) // every second ticker
	defer ticker.Stop()

	fmt.Println("start")

	for {
		select {
		case now := <-ticker.C:
			// file read with ioutil
			data, err := ioutil.ReadFile("./cpu.txt")
			if err != nil {
				fmt.Println("can't open file")
			}
			//fmt.Printf("%v, %T", data, data)
			fmt.Println("+++\n", string(data))

			cpu := string(data)

			// reflect.TypeOf() -> check type
			fmt.Println("string: ", cpu)

			// remove null char and convert string to int type
			cpu_trim := strings.TrimRight(cpu, "\n")
			cpu_usage, err := strconv.Atoi(cpu_trim)
			if err != nil {
				fmt.Println("failure cpu usage analytics")
			}

			fmt.Println("int: ", cpu_usage)

			cpu_percent <- cpu_usage
			
			fmt.Println(now.Format(time.RFC3339))
		}
	}
}
