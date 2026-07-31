//go:build windows

package proc_test

// helperOS runs the helper modes that only exist on some platforms. There are none on
// Windows, which reports neither termination by a signal nor a missing execute
// permission.
func helperOS(_ string) int {
	return 0
}
