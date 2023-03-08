package nsenter

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>

//__attribute__类似于构造函数，一旦这个包被引用，这个函数就会自动执行
__attribute__((constructor)) void enter_namespace(void) {
	char *mydocker_pid;
	//从环境变量中获取需要进入的PID
	mydocker_pid = getenv("mydocker_pid");
	if (mydocker_pid) {
		//fprintf(stdout, "got mydocker_pid=%s\n", mydocker_pid);
	} else {
		//fprintf(stdout, "missing mydocker_pid env skip nsenter");
		return;
	}
	char *mydocker_cmd;
	//从环境变量中获取需要执行的命令
	mydocker_cmd = getenv("mydocker_cmd");
	if (mydocker_cmd) {
		//fprintf(stdout, "got mydocker_cmd=%s\n", mydocker_cmd);
	} else {
		//fprintf(stdout, "missing mydocker_cmd env skip nsenter");
		return;
	}
	int i;
	char nspath[1024];
	//需要进入的五种Namespace
	char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };

	for (i=0; i<5; i++) {
		//拼接对应路径
		sprintf(nspath, "/proc/%s/ns/%s", mydocker_pid, namespaces[i]);
		int fd = open(nspath, O_RDONLY);
		//调用setns进入对应的Namespace
		if (setns(fd, 0) == -1) {
			//fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
		} else {
			//fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
		}
		close(fd);
	}
	//在进入的Namespace中执行指定的命令
	int res = system(mydocker_cmd);
	exit(0);
	return;
}
*/
import "C"
