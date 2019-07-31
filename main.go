package main

import (
	"context"
	"fmt"
	beep "github.com/gen2brain/beeep"
	"net/http"
	"os"
	"strconv"
	"time"
)

type PomadoroRatio struct {
	Work     string
	Break    string
	WorkInt  int
	BreakInt int
	Current  int
}

func (pr *PomadoroRatio) GetTimeoutChangeState() int {
	temp := pr.Current
	fmt.Println("temp =- " + string(pr.Current))
	if pr.IsWorking() {
		pr.Current = pr.BreakInt
	} else {
		pr.Current = pr.WorkInt
	}
	fmt.Printf(" changeState():%d %d ", pr.WorkInt, pr.Current)
	return temp
}
func (pr *PomadoroRatio) IsWorking() bool {
	return pr.WorkInt == pr.Current
}
func (pr *PomadoroRatio) Notify() *PomadoroRatio {
	if pr.IsWorking() {
		Notify("Work!", "Mono-tasking time!", 1, 1000)
	} else {
		Notify("Break!", "Facebook time!", 1, 1000)
	}
	return pr
}
func NewRatio(workTime string, breakTime string) PomadoroRatio {
	w, err1 := strconv.Atoi(workTime)
	b, err2 := strconv.Atoi(breakTime)
	if err1 != nil || err2 != nil {
		panic("Cannot create ratio!")
	}
	return PomadoroRatio{Break: breakTime, Work: workTime, WorkInt: w, BreakInt: b, Current: w}
}
func GetPort() string {
	port := "7765"
	if len(os.Args) == 3 && os.Args[1] == "-p" {
		port = os.Args[2]
	}
	return port
}

func Notify(title string, body string, loop int, delay int) {
	for i := 0; i < loop; i++ {
		err := beep.Alert(title, body, "tomato.png")
		if err != nil {
			panic(err)
		}
		// time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}
func Clock(stopChan chan bool, ratio PomadoroRatio, timeUnit time.Duration) {
	go func() {
		fmt.Println("starting clock")
		for {
			select {
			case <-stopChan:
				fmt.Println("stopped!")
				return
			default:
				fmt.Println("case hit")
				time.Sleep(time.Duration(ratio.Notify().GetTimeoutChangeState()) * timeUnit)
			}
		}
	}()
}

type Server struct {
	CurrentChan  chan bool
	CurrentRatio PomadoroRatio
	Port         string
	Listener     *http.Server
}

func (server *Server) Listen() {
	server.Listener = &http.Server{Addr: ":" + server.Port}
	if err := server.Listener.ListenAndServe(); err != nil {
		fmt.Println("HTTP Server Error - ", err)
	}
}
func (server *Server) Close() {
	server.CurrentChan <- true
	close(server.CurrentChan)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	server.Listener.Shutdown(ctx)
	Notify("Status", "server closed!", 1, 1000)
}
func (server *Server) InitHandlers() {
	http.HandleFunc("/stop", func(res http.ResponseWriter, req *http.Request) {
		server.Close()
	})
	http.HandleFunc("/status", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "Status!")
	})
	http.HandleFunc("/restart", func(res http.ResponseWriter, req *http.Request) {
		// stop old Server
		server.CurrentChan <- true
		close(server.CurrentChan)
		server.CurrentChan = make(chan bool)
		var b = req.FormValue("break")
		var w = req.FormValue("work")
		server.CurrentRatio = NewRatio(w, b)
		fmt.Fprintf(res, "Changed, work: "+b+" work : "+w)
		Clock(server.CurrentChan, server.CurrentRatio, time.Second)
	})
	http.HandleFunc("/start", func(res http.ResponseWriter, req *http.Request) {
		if server.CurrentChan != nil {
			return
		}
		server.CurrentChan = make(chan bool)
		var b = req.FormValue("break")
		var w = req.FormValue("work")
		fmt.Fprintf(res, "Changed, work: "+b+" work : "+w)
		pr := NewRatio(w, b)
		Clock(server.CurrentChan, pr, time.Second)
	})
}

func main() {
	port := GetPort()
	server := Server{CurrentChan: nil, CurrentRatio: PomadoroRatio{}, Port: port}
	server.InitHandlers()
	server.Listen()
}

//curl -d 'break=22&work=33' -H "Type:application/x-www-form-urlencoded" -X POST http://localhost:7765/restart
//curl -d 'break=17&work=11' -H "Type:application/x-www-form-urlencoded" -X POST http://localhost:2223/stop
//curl -d 'break=10&work=9' -H "Type:application/x-www-form-urlencoded" -X POST http://localhost:2223/start
