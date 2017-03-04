package common

// Must ensure no error happens
func Must(err error) {
	if err != nil {
		panic(err)
	}
}
