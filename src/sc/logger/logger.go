
package logger

import (
	"os"
	"time"
	"strings"
	"fmt"

	"sc/errors"

)

func timePrefix() string {
	d := time.Now()
	return fmt.Sprintf("[ %v.%02d.%02d %02d:%02d:%02d ] ", d.Year(), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second())
}

var stdout_chan chan string = make(chan string, 100)
var f_stdout, f_stderr *os.File

func outputer() {
    for {
    	s := <-stdout_chan
	    f_stdout.Write([]byte(timePrefix() + s + "\n"))
    }
}

func String(s string) {
	stdout_chan <- s
}

func Error(err *errors.Error) {

	var s = err.ErrorWithStack("generated") + errors.New(nil, 3).ErrorWithStack("dropped")
	lines := strings.Split(s, "\n")

	s = ""
	prefix := timePrefix()

	for _, line := range lines {
		s += prefix + line + "\n"
	}

	f_stderr.Write([]byte(s))

}


func Init(path string) (err error) {

    f_stdout, err = os.OpenFile(path + "stdout.log",   os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)

    if err != nil {
    	return
    }

    f_stderr, err = os.OpenFile(path + "stderr.log",   os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)

    if err != nil {
    	return
    }

	go outputer()

	return
}