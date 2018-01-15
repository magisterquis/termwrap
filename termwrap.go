// Termwrap wraps stdio in a terminal where output doesn't interefere with
// input
package main

/*
 * termwrap.go
 * Wrap stdio in a less frustrating terminal
 * By J. Stuart McMurray
 * Created 20180114
 * Last Modified 20180114
 */

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/hashicorp/go-immutable-radix"

	"golang.org/x/crypto/ssh/terminal"
)

// WORDLIST holds the autocomplete callback's list of words
var WORDLIST *iradix.Tree

// ERRORCHAN receives errors which terminate the program
var ERRORCHAN chan<- error

func main() {
	var (
		prompt = flag.String(
			"p",
			"",
			"Locally displays `prompt` at the beginning of every "+
				"input line",
		)
		aFile = flag.String(
			"t",
			"",
			"If set, uses the lines of `file` as a list of "+
				"tab-complete words",
		)
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage %v [options] command [arg [arg...]]

Runs the command, and wraps its stdio in a terminal which keeps input and
output separate.

Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Make sure we actually have args */
	if 0 == flag.NArg() {
		fmt.Fprintf(os.Stderr, "Need a command, please.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	/* Set up autocomplete list */
	if "" != *aFile {
		if err := parseAList(*aFile); nil != err {
			fmt.Fprintf(
				os.Stderr,
				"Unable to parse autocomplete file: %v",
				err,
			)
			os.Exit(5)
		}
	}

	/* Stdin should probably be a terminal. */
	infd := int(os.Stdin.Fd())
	if !terminal.IsTerminal(infd) {
		fmt.Fprintf(os.Stderr, "Warning: stdin isn't a tty.\n")
	}

	/* Set stdin to raw mode, wrap stdio */
	ps, err := terminal.MakeRaw(infd)
	if nil != err {
		fmt.Fprintf(
			os.Stderr,
			"Unable to set stdin to raw mode: %v\n",
			err,
		)
		os.Exit(2)
	}

	/* Restore terminal when we're done */
	defer func() {
		if err := terminal.Restore(infd, ps); nil != err {
			fmt.Fprintf(
				os.Stderr,
				"Unable to restore stdin: %v\r\n",
				err,
			)
			os.Exit(3)
		}
	}()

	/* Error chanel */
	ech := make(chan error)
	ERRORCHAN = ech

	/* Wrap stdio in a terminal */
	t := terminal.NewTerminal(
		struct {
			io.Reader
			io.Writer
		}{
			os.Stdin,
			os.Stdout,
		},
		*prompt,
	)
	t.AutoCompleteCallback = autoCompleteCallback

	/* Start program, hook up stdio */
	c := exec.Command(flag.Arg(0), flag.Args()[1:]...)
	c.Stdout = t
	c.Stderr = t
	in, err := c.StdinPipe()
	if nil != err {
		fmt.Fprintf(
			os.Stderr,
			"Unable to get child's stdin: %v\r\n",
			err,
		)
		os.Exit(4)
	}

	/* Proxy input */
	go func() {
		for {
			/* Get a line */
			l, err := t.ReadLine()
			if nil != err {
				ERRORCHAN <- err
				return
			}
			/* Send it to the child */
			if _, err := in.Write([]byte(l + "\n")); nil != err {
				ERRORCHAN <- err
				return
			}
		}
	}()

	/* Start child */
	go func() {
		ERRORCHAN <- c.Run()
	}()

	/* Wait for something */
	if err := <-ech; nil != err && io.EOF != err {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\r\n", err)
	}
}

/* parseAList reads the lines from fn and turns them into a radix tree for
the autocomplete callback */
func parseAList(fn string) error {
	WORDLIST = iradix.New()

	/* Slurp file */
	b, err := ioutil.ReadFile(fn)
	if nil != err {
		return err
	}

	/* Add each line to the tree */
	for _, line := range strings.Split(string(b), "\n") {
		l := strings.TrimSpace(line)
		if "" == l {
			continue
		}
		WORDLIST, _, _ = WORDLIST.Insert([]byte(l), nil)
	}

	return nil
}

/* autoCompleteCallback provides autocompletion for the terminal */
func autoCompleteCallback(
	line string,
	pos int,
	key rune,
) (newLine string, newPos int, ok bool) {
	switch key {
	case 0x03: /* Ctrl+C */
		/* Quit if it's an empty line */
		if "" == line {
			ERRORCHAN <- fmt.Errorf("keyboard interrupt")
			return "", 0, false
		}
		/* Clear the line otherwise */
		return "", 0, true
	case '\t': /* Tab, for autocomplete */
		break
	default:
		return "", 0, false
	}

	/* If we have no set of completion words, give up */
	if nil == WORDLIST {
		return "", 0, false
	}

	/* Get word on which tab was called */
	start := pos - 1
	if 0 > start {
		start = 0
	}
	for ; 0 < start && !unicode.IsSpace(rune(line[start-1])); start-- {
	}
	word := line[start:pos]

	/* Find matches */
	ms := []string{}
	WORDLIST.Root().WalkPrefix(
		[]byte(word),
		func(k []byte, v interface{}) bool {
			ms = append(ms, string(k))
			return false
		},
	)

	/* If there's no matches, do nothing */
	if 0 == len(ms) {
		return "", 0, false
	}

	/* Find the longest common prefix */
	lcp := longestCommonPrefix(ms)

	/* If there isn't one, line's unchanged */
	if 0 == len(lcp) {
		return "", 0, false
	}

	/* Put the prefix into the line */
	left := line[:start]
	right := line[pos:]
	return left + lcp + right, pos + (len(lcp) - len(word)), true

}

/* longestCommonPrefix finds the longest prefix shared between the strings. */
func longestCommonPrefix(ss []string) string {
	/* If there was no match, there's no prefix */
	if 0 == len(ss) {
		return ""
	}
	/* If we have only one match, it's the answer */
	if 1 == len(ss) {
		return ss[0]
	}

	/* Find the min and max strings */
	min := ss[0]
	max := ss[0]
	for _, s := range ss[1:] {
		if s > max {
			max = s
		} else if s < min {
			min = s
		}
	}

	/* The common prefix between the minimum and maximum strings is the
	prefix common to all strings */
	pref := ""
	for i := 0; len(min) > i && len(max) > i; i++ {
		if min[i] != max[i] {
			break
		}
		pref += string(min[i])
	}
	return pref
}
