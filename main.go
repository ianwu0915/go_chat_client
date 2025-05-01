package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func handleRecieve(done chan struct{}, once *sync.Once, conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		select {
		case <-done:
			fmt.Println("Shutting down reciever Goroutine")
			return
		default:
			// read message from reader
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Server disconnected:", err)
				once.Do(func ()  {
					close(done)
					conn.Close()
				})
				os.Exit(0)
				return 

			}
			fmt.Print(msg)
		}
	}
}

func handleCloseSignal(done chan struct{}, once *sync.Once, conn net.Conn) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("Interrupt Signal detected. Shutting down...")
	once.Do(func ()  {
		close(done)
		conn.Close()
		os.Exit(0)	
	})
}

func handleInput(done chan struct{}, once *sync.Once, conn net.Conn) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type your messages (Ctrl + C to exit):")
	for scanner.Scan() {
		select {
		case <-done:
			fmt.Println("Shutting down input handler")
			return
		default:
			message := scanner.Text()
			if message == "" {
				continue
			}

			_, err := conn.Write([]byte(message + "\n"))
			if err != nil {
				fmt.Println("Failed to send message", err)
				once.Do(func ()  {
					close(done)
					conn.Close()
				})
				os.Exit(1)
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error", err)
	}
	once.Do(func() {
		close(done)
	})
}

func main() {
	done := make(chan struct{})

	var once sync.Once

	// 1. 連接到 TCP server（使用 net.Dial）
	//    - 如果連不上，印出錯誤並退出程式
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Connection Failed: ", err)
		os.Exit(1)
	} else {
		fmt.Println("Successfully Connected to the server!")
	}

	// defer conn.Close()

	// 持續用<-sigChan 來監聽並處理系統送來的 Ctrl + c or termination
	go handleCloseSignal(done, &once, conn)

	// 2. 建立一個 goroutine：
	//    - 持續從 conn 讀取 server 廣播過來的訊息
	//    - 每次讀到，就印出來（記得換行）
	//    - 如果讀不到，可能 server 已關閉，結束 goroutine

	go handleRecieve(done, &once, conn)

	// 3. 在主線程中：
	//    - 用 bufio.NewScanner(os.Stdin) 持續讀取使用者輸入
	//    - 每行輸入後，透過 conn.Write() 發送到 server
	//    - 若輸入為空或 EOF（如 Ctrl+D），退出程式

	handleInput(done, &once, conn)

}
