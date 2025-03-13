package db

// import (
// 	"fmt"
// 	"time"
// )

// // a very basic implementation and doesn't support much,where the hell is io-multiplexing???

// type EventLoop struct {
// 	syncTasks  []func()
// 	asyncTasks []chan func()
// }

// func NewEventLoop() *EventLoop {
// 	return &EventLoop{
// 		syncTasks:  make([]func(), 0),
// 		asyncTasks: make([]chan func(), 0),
// 	}
// }

// func (e *EventLoop) AddSyncTask(task func()) {
// 	if task == nil {
// 		return
// 	}
// 	e.syncTasks = append(e.syncTasks, task)
// }

// func (e *EventLoop) AddAsyncTask(callback func()) {
// 	if callback == nil {
// 		return
// 	}
// 	ch := make(chan func(), 1)
// 	ch <- callback
// 	e.asyncTasks = append(e.asyncTasks, ch)
// }

// func (e *EventLoop) processSyncTasks() {
// 	for len(e.syncTasks) > 0 {
// 		task := e.syncTasks[0]
// 		e.syncTasks = e.syncTasks[1:]
// 		task()
// 	}
// }

// func (e *EventLoop) processAsyncTasks() {
// 	if len(e.asyncTasks) > 0 {
// 		for len(e.asyncTasks) > 0 {
// 			ch := e.asyncTasks[0]
// 			e.asyncTasks = e.asyncTasks[1:]

// 			select {
// 			case callback := <-ch:
// 				callback()
// 			default:
// 			}
// 		}
// 	}
// }

// func (e *EventLoop) Run() {
// 	for {

// 		e.processSyncTasks()
// 		e.processAsyncTasks()

// 		if len(e.syncTasks) == 0 && len(e.asyncTasks) == 0 {
// 			fmt.Println("Event loop finished, no tasks left.")
// 			break
// 		}
// 		time.Sleep(100 * time.Millisecond) // Sleep for a short time to avoid busy-waiting and save resources from unnecessary consumption.
// 	}
// }

// // func MakeGetRequest() {
// // 	fmt.Println("Starting async GET request...")

// // 	resp, err := http.Get("https://jsonplaceholder.typicode.com/todos/1")
// // 	if err != nil {
// // 		fmt.Printf("Failed to make GET request: %v\n", err)
// // 		return
// // 	}
// // 	defer resp.Body.Close()

// // 	body, err := io.ReadAll(resp.Body)
// // 	if err != nil {
// // 		fmt.Printf("Failed to read response body: %v\n", err)
// // 		return
// // 	}

// // 	fmt.Printf("GET request successful! Response: %s\n", body)
// // }

// // func main() {
// // 	loop := NewEventLoop()

// // 	loop.AddSyncTask(func() {
// // 		fmt.Println("Executing synchronous task 1")
// // 	})
// // 	loop.AddAsyncTask(func() {
// // 		MakeGetRequest()
// // 	})

// // 	loop.AddSyncTask(func() {
// // 		fmt.Println("Executing synchronous task 2")
// // 	})

// // 	loop.Run()

// // 	fmt.Println("Program finished.")
// // }
