package main

import (
	"fmt"
	"bufio"
	"os"
	"github.com/marella/godb/godb"
)

func main() {
	g := godb.New("127.0.0.1", "db_name")
	s := "start"
	scanner := bufio.NewScanner(os.Stdin)
	for s != "quit" {
		fmt.Print("$: ")
		scanner.Scan() // read input from stdin
		s = scanner.Text()
		if err := scanner.Err(); err != nil {
			break
		}

		r, ok := g.Query(s)
		if !ok {
			// 
		}
		fmt.Println(r)
	}
}