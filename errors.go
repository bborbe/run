package run

import "bytes"

type ErrorList []error

func NewErrorList(errors ...error) ErrorList {
	return ErrorList(errors)
}

func NewErrorListByChan(errors <-chan error) ErrorList {
	var list []error
	for err := range errors {
		list = append(list, err)
	}
	return NewErrorList(list...)
}

func (e ErrorList) Error() string {
	buf := bytes.NewBufferString("errors: ")
	first := true
	for _, err := range e {
		if first {
			first = false
		} else {
			buf.WriteString(", ")
		}
		buf.WriteString(err.Error())
	}
	return buf.String()
}
