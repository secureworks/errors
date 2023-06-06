package errors_test

import (
	"fmt"
	"github.com/secureworks/errors"
	"io"
	"os"
)

func openCustomerFile(customerID string) (*os.File, error) {
	f, err := os.Open("/data/customers/" + customerID + ".csv")
	if err != nil {
		return nil, errors.Chain("failed to open customer file", err)
	}
	return f, nil
}

func readCustomerInfo(customerID string) (string, error) {
	f, err := openCustomerFile(customerID)
	if err != nil {
		return "", errors.Chain("failed to open customer", err)
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return "", errors.Chain("failed to read customer info", err)
	}
	return string(bytes), nil
}

func Example() {
	info, err := readCustomerInfo("arik")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "%+v", err)
	}
	_, _ = fmt.Fprintf(os.Stdout, info)

	// Output:
	// failed to open customer
	//      github.com/secureworks/errors_test.readCustomerInfo
	//      	/Users/arikkfir/Development/github.com/arik-kfir/errors/chain_test.go:21
	//      github.com/secureworks/errors_test.Example
	//      	/Users/arikkfir/Development/github.com/arik-kfir/errors/chain_test.go:33
	//      testing.runExample
	//      	/opt/homebrew/opt/go/libexec/src/testing/run_example.go:63
	//      testing.runExamples
	//      	/opt/homebrew/opt/go/libexec/src/testing/example.go:44
	//      testing.(*M).Run
	//      	/opt/homebrew/opt/go/libexec/src/testing/testing.go:1908
	//      main.main
	//      	_testmain.go:221
	//      runtime.main
	//      	/opt/homebrew/opt/go/libexec/src/runtime/proc.go:250
	//
	// CAUSED BY: failed to open customer file
	//      github.com/secureworks/errors_test.openCustomerFile
	//      	/Users/arikkfir/Development/github.com/arik-kfir/errors/chain_test.go:13
	//      github.com/secureworks/errors_test.readCustomerInfo
	//      	/Users/arikkfir/Development/github.com/arik-kfir/errors/chain_test.go:19
	//      github.com/secureworks/errors_test.Example
	//      	/Users/arikkfir/Development/github.com/arik-kfir/errors/chain_test.go:33
	//      testing.runExample
	//      	/opt/homebrew/opt/go/libexec/src/testing/run_example.go:63
	//      testing.runExamples
	//      	/opt/homebrew/opt/go/libexec/src/testing/example.go:44
	//      testing.(*M).Run
	//      	/opt/homebrew/opt/go/libexec/src/testing/testing.go:1908
	//      main.main
	//      	_testmain.go:221
	//      runtime.main
	//      	/opt/homebrew/opt/go/libexec/src/runtime/proc.go:250
	//
	// CAUSED BY: open /data/customers/arik.csv: no such file or directory
	//
}
