package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/mbedford-stream/mbgofuncs/mbfile"
	"github.com/mbedford-stream/mbgofuncs/mbrandom"

	"github.com/fatih/color"
)

// Global vars just in case we need to pass them around
var green = color.New(color.FgGreen).SprintfFunc()
var red = color.New(color.FgRed).SprintfFunc()
var yellow = color.New(color.FgYellow).SprintfFunc()
var hostsFile string

func main() {
	if runtime.GOOS == "windows" {
		fmt.Printf("%s\n", red("I know this breaks your heart, but this will not work on Windows"))
		os.Exit(0)
	}

	var undoFile bool
	flag.BoolVar(&undoFile, "undo", false, "Restores from most recent backup")
	flag.Parse()

	// Get current user home directory
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(red("Can't detect home directory"))
		os.Exit(0)
	}
	hostsFile = homedir + "/.ssh/known_hosts"

	// confirm we find the known_hosts file where we expect to and alert if its not there
	if !mbfile.FileExistsAndIsNotADirectory(hostsFile) {
		fmt.Printf("%s %s\n", red("Could not locate hosts file: "), hostsFile)
		os.Exit(0)
	}

	if undoFile {
		restoreConfirm := mbrandom.ForceSelect("Restore previous known_hosts file from backup? (y/n): ", "y", "n")
		if restoreConfirm == "y" {
			err := prevRestore()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", green("Previous file version restored"))
			os.Exit(0)
		} else {
			os.Exit(0)
		}
	}

	fixLineStr := flag.Arg(0)
	if fixLineStr == "" {
		fmt.Printf("%s\n", red("Line needing fixed cannot be blank"))
		os.Exit(0)
	}
	fixLine, err := strconv.Atoi(fixLineStr)
	if err != nil {
		fmt.Printf("Can't convert %s to a line number\n", red(fixLineStr))
		os.Exit(0)
	}

	hostsLines, err := mbfile.FileReadReturnLines(hostsFile)
	if err != nil {
		log.Fatalf("Could not read current file: %s\n%s", hostsFile, err)
	}

	if fixLine >= len(hostsLines) {
		fmt.Printf("There aren't %s lines in the file, try again\n", red(fixLineStr))
		os.Exit(0)
	}

	newFile, err := removeLine(fixLine, hostsLines)
	if err != nil {
		fmt.Println(red("%s\n", err))
		os.Exit(0)
	}

	err = mbfile.WriteListToFile(newFile, hostsFile, 0666)
	if err != nil {
		fmt.Printf("Could not create new file: %s\n%s", hostsFile, err)
		os.Exit(0)
	}

}

func removeLine(lineNum int, fileLines []string) ([]string, error) {
	var fixedLines []string
	lineFound := false
	remLine := lineNum - 1
	for k, l := range fileLines {
		if k == remLine {
			var lineHostname string
			if strings.Contains(l, "@cert-authority") {
				lineHostname = strings.Split(l, " ")[1]
			} else {
				lineHostname = strings.Split(l, " ")[0]
			}
			selectQ := fmt.Sprintln(yellow("Remove this line? : %s\n%s ", lineHostname, "(y/n)"))
			if mbrandom.ForceSelect(selectQ, "y", "n") == "y" {
				err := createBackup()
				if err != nil {
					log.Fatal(err)
				}
				lineFound = true
				continue
			} else {
				return fixedLines, errors.New("line not removed")
			}

		}
		fixedLines = append(fixedLines, l)
	}

	if !lineFound {
		return fixedLines, errors.New("line not found in file")
	}
	return fixedLines, nil
}

func prevRestore() error {
	err := mbfile.CopyFile(hostsFile, hostsFile+".pre-undo", 0666)
	if err != nil {
		return errors.New("could not create pre-undo file, exiting")
	}
	err = mbfile.CopyFile(hostsFile+".backup", hostsFile, 0666)
	if err != nil {
		return errors.New("could not restore file, exiting")
	}

	return nil
}

func createBackup() error {
	err := mbfile.CopyFile(hostsFile, hostsFile+".backup", 0666)
	if err != nil {
		return fmt.Errorf("could not create backup of current file: \n%s", err)
	}
	return nil
}
