package nmstate

// #cgo CFLAGS: -g -Wall
// #cgo LDFLAGS: -L/usr/local/lib64 -lnmstate -Wl,-rpath=/usr/local/lib64
// #include <nmstate.h>
// #include <stdlib.h>
import "C"
import "fmt"

func RetrieveNetState(flags uint) (string, error) {
	var (
		state    *C.char
		log      *C.char
		err_kind *C.char
		err_msg  *C.char
	)
	rc := C.nmstate_net_state_retrieve(C.uint(flags), &state, &log, &err_kind, &err_msg)

	defer func() {
		C.nmstate_net_state_free(state)
		C.nmstate_err_msg_free(err_msg)
		C.nmstate_err_kind_free(err_kind)
		C.nmstate_log_free(log)
	}()
	if rc != 0 {
		//TODO: Handle the logs properly
		return "", fmt.Errorf("failed retrieving nmstate net state with rc: %d and logs: %s", rc, C.GoString(log))
	}
	return C.GoString(state), nil
}

func ApplyNetState(flags uint, state string) (string, error) {
	var (
		c_state  *C.char
		log      *C.char
		err_kind *C.char
		err_msg  *C.char
	)
	c_state = C.CString(state)
	rc := C.nmstate_net_state_apply(C.uint(flags), c_state, &log, &err_kind, &err_msg)

	defer func() {
		C.nmstate_net_state_free(c_state)
		C.nmstate_err_msg_free(err_msg)
		C.nmstate_err_kind_free(err_kind)
		C.nmstate_log_free(log)
	}()
	if rc != 0 {
		//TODO: Handle the logs properly
		return "", fmt.Errorf("failed applying nmstate net state %s with rc: %d and logs: %s", state, rc, C.GoString(log))
	}
	return C.GoString(c_state), nil
}
