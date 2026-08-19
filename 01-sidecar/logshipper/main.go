package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	f, err := os.Open("../shared/webapp.log")
	if err != nil {
		fmt.Println("could not open the file")
		return
	}

	defer func() {
		_ = f.Close()
	}()

	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Println("error seeking file")
		return
	}
	buf := make([]byte, 1024)

	for {

		n, err := f.Read(buf)
		if n > 0 {
			fmt.Println(string(buf[:n]))
		}
		if err != nil {
			if err == io.EOF {
				time.Sleep(time.Second * 1)
			} else {
				fmt.Println("error reading buffer")
				return
			}
		}
	}
}
