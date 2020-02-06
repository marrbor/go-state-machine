// Tool for puml(plant uml) file to go-state-machine STT(State Transition Table).
package go_state_machine

import (
	"bufio"
	"fmt"
	"os"
)

func usage(msg string, code int) {
	if _, err := fmt.Fprintf(os.Stderr, msg); err != nil {
		errExit(err)
	}
	fmt.Println("usage: go_state_machine input-file [output-file]")
	os.Exit(code)
}

func errExit(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage("Give me input file name.", 1)
	}

	input := os.Args[1]
	fp, err := os.Open(input)
	if err != nil {
		errExit(err)
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {

	}
	if err := scanner.Err(); err != nil {
		errExit(err)
	}






	os.Exit(0)
}
