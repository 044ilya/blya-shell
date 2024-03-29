package main

/*
extern void disableRawMode();
extern void enableRawMode();
*/
import "C"

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	HISTSIZE  = 25
	HISTFILE  string
	HISTMEM   []string
	HISTCOUNT int
	HISTLINE  string
	CONFIG    string
	aliases   map[string]string
)

const (
	TOKDELIM  = " \t\r\n\a"
	ERRFORMAT = "sesh: %s\n"
)

func main() {
	blyaLoop()
}

func blyaLoop() {
	HISTMEM = initHistory(HISTMEM)
	status := 1
	reader := bufio.NewReader(os.Stdin)

	for status != 0 {
		C.enableRawMode()
		fmt.Printf("λ ", os.Getenv("CWD"), " ")
		line, discard, cursorPos, histCounter, shellEditor := "", false, 0, 0, false
		for {
			c, _ := reader.ReadByte()
			if shellEditor && c == 13 {
				line = line[:len(line)-1]
				fmt.Println()
				shellEditor = false
				cursorPos = len(line)
				continue
			}
			shellEditor = false
			if c == 27 {
				c1, _ := reader.ReadByte()
				if c1 == '[' {
					c2, _ := reader.ReadByte()
					switch c2 {
					case 'A':
						if len(HISTMEM) != 0 && histCounter < len(HISTMEM) {
							for cursorPos > 0 {
								fmt.Printf("\b\033[J")
								cursorPos--
							}
							line = strings.Split(HISTMEM[histCounter], "::")[2]
							fmt.Printf(line)
							cursorPos = len(line)
							histCounter++
						}
					case 'B':
						if len(HISTMEM) != 0 && histCounter > 0 {
							for cursorPos > 0 {
								fmt.Printf("\b\033[J")
								cursorPos--
							}
							histCounter--
							line = strings.Split(HISTMEM[histCounter], "::")[2]
							fmt.Printf(line)
							cursorPos = len(line)
						}
					case 'C':
						if cursorPos < len(line) {
							fmt.Printf("\033[C")
							cursorPos++
						}
					case 'D':
						if cursorPos > 0 {
							fmt.Printf("\033[D")
							cursorPos--
						}
					}
				}
				continue
			}
			// backspace was pressed
			if c == 127 {
				if cursorPos > 0 {
					if cursorPos != len(line) {
						temp, oldLength := line[cursorPos:], len(line)
						fmt.Printf("\b\033[K%s", temp)
						for oldLength != cursorPos {
							fmt.Printf("\033[D")
							oldLength--
						}
						line = line[:cursorPos-1] + temp
						cursorPos--
					} else {
						fmt.Print("\b\033[K")
						line = line[:len(line)-1]
						cursorPos--
					}
				}
				continue
			}
			// ctrl-c was pressed
			if c == 3 {
				fmt.Println("^C")
				discard = true
				break
			}
			// ctrl-d was pressed
			if c == 4 {
				exit()
			}
			// the enter key was pressed
			if c == 13 {
				fmt.Println()
				break
			}
			// tab was pressed
			if c == 9 {
				args := strings.Fields(line)
				if len(line) > 1 {
					arg := args[len(args)-1]
					pattern, dir := arg, "."
					if strings.Contains(arg, "/") {
						pattern, dir = filepath.Base(arg), filepath.Dir(arg)
					}
					files, _ := ioutil.ReadDir(dir)
					matches := make([]string, 0, 10)
					for _, file := range files {
						if strings.HasPrefix(file.Name(), pattern) {
							matches = append(matches, file.Name())
						}
					}
					if len(matches) == 1 {
						pathToAppend := matches[0]
						if strings.Contains(arg, "/") {
							pathToAppend = fmt.Sprintf("%s/%s", dir, matches[0])
						}
						args[len(args)-1] = pathToAppend
						line = strings.Join(args, " ")
						for cursorPos > 0 {
							fmt.Printf("\b\033[K")
							cursorPos--
						}
						fmt.Printf("%s", line)
						cursorPos = len(line)
						continue
					}
				}
				continue
			}
			if cursorPos == len(line) {
				fmt.Printf("%c", c)
				line += string(c)
				cursorPos = len(line)
			} else {
				temp, oldLength := line[cursorPos:], len(line)
				fmt.Printf("\033[K%c%s", c, temp)
				for oldLength != cursorPos {
					fmt.Printf("\033[D")
					oldLength--
				}
				line = line[:cursorPos] + string(c) + temp
				cursorPos++
			}
			if c == '\\' {
				shellEditor = true
			}
		}
		C.disableRawMode()
		if line == "" || discard {
			status = 1
			continue
		}
		HISTLINE, status = line, 1
		line = strings.Replace(line, "~", os.Getenv("HOME"), -1)
		args, ok := parseLine(line)
		if ok && args != nil {
			status = execute(args)
		} else {
			status = 2
		}
		if status == 1 {
			/* Store line in history */
			if HISTCOUNT == HISTSIZE {
				HISTMEM = HISTMEM[1:]
				HISTCOUNT = 0
			}
			HISTMEM = append([]string{HISTLINE}, HISTMEM...)
			HISTCOUNT++
		}
	}
}

func parseLine(line string) ([]string, bool) {
	args := regexp.MustCompile("'(.+)'|\"(.+)\"|\\S+").FindAllString(line, -1)
	for i, arg := range args {
		if (arg[0] == '"' && arg[len(arg)-1] == '"') || (arg[0] == '\'' && arg[len(arg)-1] == '\'') {
			args[i] = arg[1 : len(arg)-1]
		}
	}
	if args[0] == "alias" {
		if len(args) == 1 {
			fmt.Printf(ERRFORMAT, "arguments needed for alias")
			return nil, false
		}
		for _, i := range args[1:] {
			aliasArgs := strings.Split(i, "=")
			if len(aliasArgs) != 2 {
				fmt.Printf(ERRFORMAT, "wrong format of alias")
				return nil, false
			}
			aliases[aliasArgs[0]] = aliasArgs[1]
		}
		return args, false
	}
	if args[0] == "export" {
		if len(args) == 1 {
			fmt.Printf(ERRFORMAT, "argument needed for export")
			return nil, false
		}
		exportArgs := strings.Split(args[1], "=")
		if len(exportArgs) != 2 {
			fmt.Printf(ERRFORMAT, "wrong format of export")
			return nil, false
		}
		os.Setenv(exportArgs[0], exportArgs[1])
		return args, false
	}
	// replace if an alias
	for i, arg := range args {
		if val, ok := aliases[arg]; ok {
			args[i] = val
		}
	}
	// replace if an environment variable
	for i, arg := range args {
		if arg[0] == '$' {
			args[i] = os.Getenv(arg[1:])
		}
	}
	// wildcard support (not really efficient)
	wildcardArgs := make([]string, 0, 5)
	for _, arg := range args {
		if strings.Contains(arg, "*") || strings.Contains(arg, "?") {
			matches, _ := filepath.Glob(arg)
			wildcardArgs = append(wildcardArgs, matches...)
		} else {
			wildcardArgs = append(wildcardArgs, arg)
		}
	}
	args = wildcardArgs
	return args, true
}

func execute(args []string) int {
	if len(args) == 0 {
		return 1
	}
	for k, v := range builtins {
		if args[0] == k {
			timestamp := time.Now().String()
			HISTLINE = fmt.Sprintf("%d::%s::%s", os.Getpid(), timestamp, HISTLINE)
			return v(args[1:])
		}
	}
	return launch(args)
}

func initHistory(history []string) []string {
	if _, err := os.Stat(HISTFILE); err == nil {
		f, _ := os.OpenFile(HISTFILE, os.O_RDONLY, 0666)
		defer f.Close()
		/* Read file and store each line in history slice */
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			history = append(history, scanner.Text())
			HISTCOUNT++
		}
	}
	return history
}

func exit() {
	f, err := os.OpenFile(HISTFILE, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf(ERRFORMAT, err.Error())
	}
	defer f.Close()
	for _, i := range HISTMEM {
		f.Write([]byte(i))
		f.Write([]byte("\n"))
	}
	os.Exit(0)
}
