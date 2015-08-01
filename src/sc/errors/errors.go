
package errors

import (
	"runtime"
	"fmt"
	"strings"
)

var MaxStackDepth = 50

type Error struct {
	hasErr	bool
	err    error
	stack  []uintptr
	frames []StackFrame
}

func New(args ...interface{}) (*Error) {

	var hasErr bool = true
	var err error
	var offset int = 2

	for index, arg := range args {

	    switch index {

		case 0:
		    switch t := arg.(type) {
			case error:
				err = t
			case nil:
				hasErr = false
			default:
				err = fmt.Errorf("%v", t)
		    }

		case 1:
		    switch t := arg.(type) {
		    case int:
				offset = t
			}
		}

	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(offset, stack[:])

	return &Error{
		hasErr:	hasErr,
		err:	err,
		stack:	stack[:length-2],
	}
}

func (err *Error) ErrorWithStack(title string) (s string) {
	s = ""
	if err.hasErr {
		s = "Error: " + err.err.Error() + "\n"
	}
	return s + "\n" + title + "\n" + err.Stack()
}

func (err *Error) Stack() string {

	frames := err.StackFrames()
	a := make([]string, len(frames))	

	for index, frame := range frames {
		a[index] = "\n" + frame.String()
	}

	return strings.Join(a, "")
}

func (err *Error) StackFrames() []StackFrame {
	if err.frames == nil {
		err.frames = make([]StackFrame, len(err.stack))

		for i, pc := range err.stack {
			err.frames[i] = NewStackFrame(pc)
		}
	}

	return err.frames
}
