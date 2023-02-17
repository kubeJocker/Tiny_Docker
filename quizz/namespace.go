package quizz

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

/*
UTS Namespace主要用来隔离nodename和domainname两个系统标识。在UTS namespace里，每个namespace允许有自己的hostname。
系统API 中的clone()创建新的进程。根据填入的参数来判断哪些namesapce会被创建，而且它们的子进程也会被包含到这些namespace中。
*/
func namespace() {
	cmd := exec.Command("sh")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
