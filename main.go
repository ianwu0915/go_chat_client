package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	done := make(chan struct{})
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 1. 連接到 TCP server（使用 net.Dial）
	//    - 如果連不上，印出錯誤並退出程式
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Connection Failed: ", err)
	}

	defer conn.Close()

	// 持續用<-sigChan 來監聽並處理系統送來的 Ctrl + c or termination
	go func() {
		<-sigChan
		fmt.Println("\n[Client] Ctrl+C detected. Closing connection...")
		close(done)
		conn.Close()
		os.Exit(0)
	}()

	// 2. 建立一個 goroutine：
	//    - 持續從 conn 讀取 server 廣播過來的訊息
	//    - 每次讀到，就印出來（記得換行）
	//    - 如果讀不到，可能 server 已關閉，結束 goroutine
	go func() {
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
					close(done)
					os.Exit(0)
					return
				}
				fmt.Println(msg)
			}
		}
	}()

	// 3. 在主線程中：
	//    - 用 bufio.NewScanner(os.Stdin) 持續讀取使用者輸入
	//    - 每行輸入後，透過 conn.Write() 發送到 server
	//    - 若輸入為空或 EOF（如 Ctrl+D），退出程式

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type your messages (Ctrl + C to exit):")
	for scanner.Scan() {
		select {
		case <-done:
			fmt.Println("Shutting down reciever Goroutine")
			return
		default:
			message := scanner.Text()
			if message == "" {
				continue
			}

			_, err := conn.Write([]byte(message + "\n"))
			if err != nil {
				fmt.Println("Failed to send message", err)
				close(done)
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error", err)
	}

	close(done)

}
