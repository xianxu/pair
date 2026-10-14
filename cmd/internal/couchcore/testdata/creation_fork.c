// A standalone process can safely pause between fork and exec; Go cannot.
// fd3 is the lease, stdin commands exec, stdout acknowledges child readiness.
#include <fcntl.h>
#include <signal.h>
#include <sys/wait.h>
#include <unistd.h>

int main(void) {
    int ready[2];
    if (fcntl(3, F_SETFD, FD_CLOEXEC) < 0 || pipe(ready) < 0) return 1;
    pid_t child = fork();
    if (child < 0) return 2;
    if (child == 0) {
        close(ready[0]);
        char byte = 'R';
        if (write(ready[1], &byte, 1) != 1) _exit(3);
        close(ready[1]);
        if (read(STDIN_FILENO, &byte, 1) != 1) _exit(4);
        execl("/usr/bin/true", "true", (char *)0);
        _exit(5);
    }
    close(ready[1]);
    char byte;
    int acknowledged = read(ready[0], &byte, 1);
    close(ready[0]);
    close(3);
    if (acknowledged != 1 || write(STDOUT_FILENO, &byte, 1) != 1) {
        kill(child, SIGKILL);
        waitpid(child, 0, 0);
        return 6;
    }
    int status;
    if (waitpid(child, &status, 0) != child) return 7;
    return WIFEXITED(status) ? WEXITSTATUS(status) : 8;
}
