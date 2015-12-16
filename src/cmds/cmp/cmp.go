// Copyright 2013 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Cmp compares the two files and prints a message if the contents differ.

cmp [ –lLs ] file1 file2 [ offset1 [ offset2 ] ]

The options are:
	–l    Print the byte number (decimal) and the differing bytes (octal) for each difference.
	–L    Print the line number of the first differing byte.
	–s    Print nothing for differing files, but set the exit status.

-If offsets are given, comparison starts at the designated byte position of the corresponding file.
-Offsets that begin with 0x are hexadecimal; with 0, octal; with anything else, decimal.
*/

package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"strconv"
)

var long = flag.Bool("l", false, "print the byte number (decimal) and the differing bytes (hexadecimal) for each difference")
var line = flag.Bool("L", false, "print the line number of the first differing byte")
var silent = flag.Bool("s", false, "print nothing for differing files, but set the exit status")

func emit(rs io.ReadSeeker, c chan byte, offset int64) error {
	if offset > 0 {
		if _, err := rs.Seek(offset, 0); err != nil {
			log.Fatalf("%v", err)
		}
	}

	b := bufio.NewReader(rs)
	for {
		b, err := b.ReadByte()
		if err != nil {
			close(c)
			return err
		}
		c <- b
	}
}

func openFile(name string) (*os.File, error) {
	var f *os.File
	var err error

	if name == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(name)
	}

	return f, err
}

func main() {
	flag.Parse()
	var offset [2]int64
	var f *os.File
	var err error

	fnames := flag.Args()

	switch len(fnames) {
	case 2:
	case 3:
		offset[0], err = strconv.ParseInt(fnames[2], 0, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bad offset1: %s: %v\n", fnames[2], err)
			return
		}
	case 4:
		offset[0], err = strconv.ParseInt(fnames[2], 0, 64)
		if err != nil {
			log.Printf("bad offset1: %s: %v\n", fnames[2], err)
			return
		}
		offset[1], err = strconv.ParseInt(fnames[3], 0, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bad offset2: %s: %v\n", fnames[3], err)
			return
		}
	default:
		log.Fatalf("expected two filenames (and one to two optional offsets), got %d", len(fnames))
	}

	c := make([]chan byte, 2)

	for i := 0; i < 2; i++ {
		if f, err = openFile(fnames[i]); err != nil {
			log.Fatalf("Failed to open %s: %v", fnames[i], err)
		}
		c[i] = make(chan byte, 8192)
		go emit(f, c[i], offset[i])
	}

	lineno, charno := int64(1), int64(1)
	var b1, b2 byte
	for {
		b1 = <-c[0]
		b2 = <-c[1]

		if b1 != b2 {
			if *silent {
				os.Exit(1)
			}
			if *line {
				fmt.Fprintf(os.Stderr, "%s %s differ: char %d line %d\n", fnames[0], fnames[1], charno, lineno)
				os.Exit(1)
			}
			if *long {
				if b1 == '\u0000' {
					fmt.Fprintf(os.Stderr, "EOF on %s\n", fnames[0])
					os.Exit(1)
				}
				if b2 == '\u0000' {
					fmt.Fprintf(os.Stderr, "EOF on %s\n", fnames[1])
					os.Exit(1)
				}
				fmt.Fprintf(os.Stderr, "%8d %#.2o %#.2o\n", charno, b1, b2)
				goto skip
			}
			fmt.Fprintf(os.Stderr, "%s %s differ: char %d\n", fnames[0], fnames[1], charno)
			os.Exit(1)
		}
	skip:
		charno++
		if b1 == '\n' {
			lineno++
		}
		if b1 == '\u0000' && b2 == '\u0000' {
			os.Exit(0)
		}
	}
}
