package iperf

import (
	"context"
	"fmt"
	"github.com/Netflix/go-expect"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func ExecuteAsync(cmd string) (outPipe io.ReadCloser, errPipe io.ReadCloser, exitCode chan int, err error) {
	exitCode = make(chan int)
	cmdParts := strings.Fields(cmd)
	binary, err := exec.LookPath(cmdParts[0])
	if err != nil {
		return nil, nil, nil, err
	}
	exe := exec.Command(binary, cmdParts[1:]...)
	outPipe, err = exe.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	errPipe, err = exe.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	err = exe.Start()
	if err != nil {
		return nil, nil, nil, err
	}
	go func() {
		if err := exe.Wait(); err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					exitCode <- status.ExitStatus()
				}
			}
		} else {
			exitCode <- 0
		}
	}()
	return outPipe, errPipe, exitCode, nil
}

func ExecuteAsyncWithCancel(cmd string) (stdOut io.ReadCloser, stdErr io.ReadCloser, exitCode chan int, cancelToken context.CancelFunc, pid int, err error) {
	return ExecuteAsyncWithCancelReadIndicator(cmd, nil)
}

func ExecuteAsyncWithCancelReadIndicator(cmd string, readIndicator chan interface{}) (stdOut io.ReadCloser, stdErr io.ReadCloser, exitCode chan int, cancelToken context.CancelFunc, pid int, err error) {
	return executeAsyncWithCancel(cmd, readIndicator)
}

func executeAsyncWithCancel(cmd string, readIndicator chan interface{}) (stdOut io.ReadCloser, stdErr io.ReadCloser, exitCode chan int, cancelToken context.CancelFunc, pid int, err error) {
	exitCode = make(chan int)
	ctx, cancel := context.WithCancel(context.Background())
	cmdParts := strings.Fields(cmd)
	binary, err := exec.LookPath(cmdParts[0])
	if err != nil {
		defer cancel()
		return nil, nil, nil, nil, -1, err
	}

	exe := exec.CommandContext(ctx, binary, cmdParts[1:]...)
	isRsaConnection := strings.Contains(cmd, "--rsa-private-key-file")
	var c *expect.Console
	if isRsaConnection {
		c, err := expect.NewConsole(expect.WithStdout(os.Stdout))
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()
		exe.Stdin = c.Tty()
		exe.Stdout = c.Tty()
		exe.Stderr = c.Tty()
	} else {
		stdOut, err = exe.StdoutPipe()
		if err != nil {
			defer cancel()
			return nil, nil, nil, nil, -1, err
		}
		stdErr, err = exe.StderrPipe()
		if err != nil {
			defer cancel()
			return nil, nil, nil, nil, -1, err
		}

	}
	err = exe.Start()
	if err != nil {
		defer cancel()
		return nil, nil, nil, nil, -1, err
	}
	log.Printf("executing: %s\n", exe.String())

	if isRsaConnection {
		_, err = c.ExpectString("Password:")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(c, "%s\n", cmd)
	}
	go func() {
		// Note: Wait() will close the Stdout/Stderr and in some cases can do it before we read. In order to prevent
		// this we need to actually wait until the caller has finished reading.
		if readIndicator != nil {
			<-readIndicator
		}
		if err := exe.Wait(); err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					exitCode <- status.ExitStatus()
				}
			}
		} else {
			exitCode <- 0
		}
		if isRsaConnection {
			eof, err := c.Expect(expect.WithTimeout(1 * time.Second))
			if err != nil {
				log.Println("Expect WithTimeout Error:", err)
				return
			}
			stdOut = io.NopCloser(strings.NewReader(eof))
		}
	}()
	return stdOut, stdErr, exitCode, cancel, exe.Process.Pid, nil
}
