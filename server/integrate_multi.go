package main
 
import (
	"fmt"
	"os"
	"net"
	"math"
	"time"
	"sync"
	"io/ioutil"
	"strings"
	"strconv"
	"os/signal"
	"os/exec"
	"syscall"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type requestCache struct {
	resp	http.ResponseWriter
	req		*http.Request
}

var (
	mutex sync.Mutex
	cmd *exec.Cmd
	cpu_percent = make(chan int)
	next_req [4]chan int

	index int 
	sig os.Signal
	
	// parameters for evaluation
	server_num = make([]int, 0)
	parse_req = make([]int, 0)
	self_req = make([]int, 0)
	neighbor_req = make([]int, 0)
	index_parse int 
	index_add int
	index_remove int

	flag bool = false

	queue []requestCache
	csvData strings.Builder

	// set proxy IP and port
	proxyIPs = [4]string{"10.10.20.2", "10.10.30.2", "10.10.40.2", "10.10.50.2"}	
)

const (
	kappa float64 = 0.2 // diffusion coefficient (const)
	threshold int = 50

	mackerel int = 4 // server数
	
	SECOND time.Duration = 1
	MILLISECOND time.Duration = 100

	tcp_port string = ":8000"
	udp_port string = ":8001"
	udp_port_serv string = ":8001"

	
	proxyPort string = "8002"	
)

// copy and paste
// time.Sleep(SECOND * time.Second)
// time.Sleep(MILLISECOND * time.Millisecond)

func init() {
	cmd = exec.Command("sh", "cpu.sh")
	fmt.Println("123")
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	for i := range next_req {
		next_req[i] = make(chan int)
	}
	fmt.Println(next_req)

	fmt.Println("get")
}

func main() {
	// variant setting
	c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	fmt.Printf("%T \n", c)
 
	// 4 process running
 
	// every 0.1 s
	go ReadCPU_usage()
 
	// 
	go HTTPServer()

	// every 0.1 s
	go GetFeedback()
 
	// 
	go Socket_Server()
 
	// every 0.1s
	go Background()
	
	// temporary for debug
	for {
		select {
		case sig = <-c:
			fmt.Println("process finish")
			final()
			cmd.Process.Kill()
			return
		default:
			cpu := <-cpu_percent

			next_index := make([]int, mackerel)
			for n := range next_req {
				next_index[n] = <- next_req[n]
			}
			//next_index := <- next_req
			fmt.Println("neighbor len(req):", cpu , next_index)
			go RequestProcessor()
		
			if cpu > threshold {
				// support for multiple clients
				flag = true
				//time.Sleep(100 * time.Millisecond)
			}
		}
	}
	
}

// save the evaluation item to a file
func final() {
	csvData.WriteString("server_num, self_req, neighbor_req, parse_req\n")
	fmt.Println("aaa")

	maxLen := len(self_req)
	if len(neighbor_req) > maxLen {
		maxLen = len(neighbor_req)
	}
	if len(parse_req) > maxLen {
		maxLen = len(parse_req)
	}

	serverNumColumns := make(map[int][]string)
	for i := 0; i < len(server_num); i++ {
		serverNumColumns[server_num[i]] = make([]string, maxLen)
	}

	for i := 0; i < maxLen; i++ {
		serverVal := "0"
		selfVal := "0"
		neighborVal := "0"
		parseVal := "0"

		if i < len(server_num) {
			serverVal = strconv.Itoa(server_num[i])
		}
		if i < len(self_req) {
			selfVal = strconv.Itoa(self_req[i])
		}
		if i < len(neighbor_req) {
			neighborVal = strconv.Itoa(neighbor_req[i])
		}
		if i < len(parse_req) {
			parseVal = strconv.Itoa(parse_req[i])
		}

		/*
		row := serverVal + "," + selfVal + "," + neighborVal + "," + parseVal + "\n"
		csvData.WriteString(row)
		*/

		serverIndex := server_num[i]
    	serverNumColumns[serverIndex][i] = serverVal + "," + selfVal + "," + neighborVal + "," + parseVal + "\n"
	}

	for _, columns := range serverNumColumns {
		if columns[1] != "0" || columns[2] != "0" || columns[3] != "0" {
			//row := strings.Join(columns, ",") + "\n"
			for _, column := range columns {
				csvData.WriteString(column)
			}
		}
	}

	if err := ioutil.WriteFile("example.csv", []byte(csvData.String()), 0644); err != nil {
		fmt.Println("cannot create file")
	}


	fmt.Println("bbb")
 
	fmt.Println("final")
	fmt.Println("kappa: ", kappa)
	fmt.Printf("requests: all %d, current %d, response %d parse %d\n", index_add, len(queue), index_remove, index_parse)
    fmt.Printf("parse: %d\n", parse_req)
}

// check whether to proxy
func Background() {
	for {
		//fmt.Println("+++ bbb +++ ---")
		if flag == true {
			fmt.Println("--- reverseProxy!!! ---")
			// multiple server
			next_index := make([]int, mackerel)
			for i := range next_req {
				next_index[i] = <- next_req[i]
				Calculate(next_index[i], i, proxyIPs[i])
			}
			/*
			for i, addr := range proxyIPs {
				next_index[i] = <-next_req[i]
				Calculate(next_index[i], addr)
			}
			*/
			//flag = false
			// next_index := <-next_req
			// Calculate(next_index)
		} else {
			fmt.Println("not proxied")
			//time.Sleep(1 * time.Second)
			time.Sleep(MILLISECOND * time.Millisecond)
		}
	}
}

func HTTPServer() {
	// http handle func
	http.HandleFunc("/", handleRequest) 

	// HTTP Server 
	fmt.Printf("HTTP server is listening on %s...\n", tcp_port)
	http.ListenAndServe(tcp_port, nil)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	index++

	closeNotifyCh := w.(http.CloseNotifier).CloseNotify()

	select {
	case <-closeNotifyCh:
	// Handles cases where the client closes the connection
		fmt.Println("Client closed connection")
		return
	default:
		cache := requestCache {
			resp:   w,
			req:    r,
		}
		queue = append(queue, cache)
		index_add++
		fmt.Println("Request queued: ", index, len(queue))
	}
	
	time.Sleep(60 * time.Second)
	//time.Sleep(500 * time.Millisecond)
}

func RequestProcessor() {
	if len(queue) > 0 {
		mutex.Lock()
		res := queue[0].resp
		queue = queue[1:]
		index--
		index_remove++

		fmt.Println("Processing request:", index, len(queue))
		mutex.Unlock()

		res.Header().Set("Content-Type", "text/plain") // set response Content-Type
		fmt.Fprintf(res, "This is a sample response. ") // Write to ResponseWriter

		time.Sleep(MILLISECOND * time.Millisecond)
	} else {
		fmt.Println("not response")
		time.Sleep(MILLISECOND * time.Millisecond)
	}
}

func Calculate(next_index int, num int, ip_addr string) {
	index := len(queue)
	server_num = append(server_num, num)
	self_req = append(self_req, index)
	neighbor_req = append(neighbor_req, next_index)

	if index > next_index {
		//time.Sleep(100 * time.Millisecond)
		time.Sleep(SECOND * time.Second)
		fmt.Println("--- ---")
		mutex.Lock()
		
		diff := index - next_index
		parse_f := kappa * float64(diff)
		parse := int(math.Round(parse_f))
		fmt.Println(parse_f, parse)

		queue_sub := make([]requestCache, 0)
		fmt.Println("main_before: ", len(queue))

		num := len(queue) - parse
		if num < 0 {
			num = 0
		}
		queue_sub = append(queue_sub, queue[num:]...)
		queue = queue[:num]

		fmt.Println("main_after: ", len(queue))

		mutex.Unlock()
		flag = false

		// Proxy based on the number of parse
		fmt.Println("+++ aaa +++")
		reverseProxy(queue_sub, ip_addr)

	} else {
		// connect to next neighbor server (nothing to do here)
		fmt.Println("Unable to parse request")
		parse_req = append(parse_req, 0)
		flag = false // test
	}
}

func reverseProxy(queue_sub []requestCache, ip_addr string) {

	proxyURL := &url.URL {
		Scheme: "http",
		Host: ip_addr + ":" + proxyPort,
	}

	// make reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)

	fmt.Println("gogogo")
	num := 0

	for i := 0; i < len(queue_sub); i++ {
		w := queue_sub[i].resp
		r := queue_sub[i].req
		index_parse++

		proxy.ServeHTTP(w, r)
		fmt.Println("parse num:", i)
		num = i

	}
	parse_req = append(parse_req, num)
	//flag = false

	// sleep 
	time.Sleep(MILLISECOND * time.Millisecond)
}


func GetFeedback() {
	fmt.Println("feedback")
	for {
		// multiple server
		fmt.Println("get started")
		
		next := make([]int, mackerel)
		for i, addr := range proxyIPs {
			fmt.Println("search", i)
			data := 0
			next[i] = Socket_Client(data, addr)
			//fmt.Println(next[i])
			fmt.Println("fin search")
		}
		// ここまでOK
		fmt.Println(next)

		for n := range next_req {
			index := n
			go func(data int, ch chan<- int) {
				ch <- data
			}(next[index], next_req[index])
		}
		fmt.Println(next_req)
		

		//Socket_Client(next_req)
		//fmt.Println("aaa")
		time.Sleep(MILLISECOND * time.Millisecond)
		
	}
}
 
func Socket_Server() {
	// waiting connection from src container, send feedback to src container
	// set own [:Port]
	//service := ":8001"
 
	// resolve UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", udp_port)
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
 
  	fmt.Println("Server is listening on", udp_port)
 
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
		fmt.Println("<<<: ", str, n) // case success: please 6
 
		// send own number of requests to client
 
		/*
		// case CPU Usage: 
		cpu := <- cpu_percent
		cpu_str := strconv.Itoa(cpu)
		buf := []byte(cpu_str)
		*/
 
		// case number of requests that neighbor servers have:
		index := len(queue)
		req_str := strconv.Itoa(index)
		buf := []byte(req_str)
 
		_, err = conn.WriteToUDP(buf, addr)
		if err != nil {
			fmt.Println("Unable to send data", err)
			continue
		}
 
		fmt.Println("sent data: ", req_str)
	}
}
 
func Socket_Client(next int, ip_addr string) int {
	// connect to neighbor container
	// set server [IP address:Port] to connect
	serverAddr := ip_addr + udp_port_serv // ":8001"
 
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
	req_data, addr, err := conn.ReadFromUDP(data)
	if err != nil {
		fmt.Println("No data:", err)
		final()
		os.Exit(1)
	}
 
	fmt.Println("---: ", req_data, addr)
 
	req_str := string(data[:req_data])
	req, err := strconv.Atoi(req_str)
	if err != nil {
		fmt.Println("failure get request", err)
	}
 
	fmt.Println(">>>: ", req, req_data) // case success: 13 2 
 
	next = req
	fmt.Println("ppp, ", next)

	return next
}
 
func ReadCPU_usage() {
	//ticker := time.NewTicker(1 * time.Second) // every second ticker
	ticker := time.NewTicker(MILLISECOND * time.Millisecond) // every millisecond
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
 
			cpu := string(data)
 
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

 