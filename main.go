package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func handleRecieve(ctx context.Context, cancel context.CancelFunc, conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down reciever Goroutine")
			return
		default:
			// read message from reader
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Server disconnected:", err)
				cancel()
				return

			}
			fmt.Print(msg)
		}
	}
}

func handleCloseSignal(cancel context.CancelFunc, conn net.Conn) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("Interrupt Signal detected. Shutting down...")

	cancel() //會通知所有的goroutine to STOP
	conn.Close()
}

func handleInput(ctx context.Context, cancel context.CancelFunc, conn net.Conn) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type your messages (Ctrl + C to exit):")
	for scanner.Scan() {
		select {
		case <-ctx.Done():
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
				cancel()
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error", err)
	}
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	go handleCloseSignal(cancel, conn)

	// 2. 建立一個 goroutine：
	//    - 持續從 conn 讀取 server 廣播過來的訊息
	//    - 每次讀到，就印出來（記得換行）
	//    - 如果讀不到，可能 server 已關閉，結束 goroutine

	go handleRecieve(ctx, cancel, conn)

	// 3. 在主線程中：
	//    - 用 bufio.NewScanner(os.Stdin) 持續讀取使用者輸入
	//    - 每行輸入後，透過 conn.Write() 發送到 server
	//    - 若輸入為空或 EOF（如 Ctrl+D），退出程式

	go handleInput(ctx, cancel, conn)

	<-ctx.Done()
	fmt.Println("Client exited.")

}
