# -*- coding: utf-8 -*-
from __future__ import unicode_literals


import os, time, logging
from collections import deque
from threading import Thread
import paramiko


from utils.printing import *
from utils.mylogging import get_logger


logger = get_logger(__name__)


# logger = logging.getLogger(__name__)
# logger.setLevel(logging.DEBUG)
# handler = logging.FileHandler(filename='sshRunner.log', mode='w', )
# handler.setFormatter(logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s'))
# logger.addHandler(handler)
# handler2 = logging.StreamHandler()
# handler2.setFormatter(logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s'))
# logger.addHandler(handler2)


def t1():
    ssh = subprocess.Popen(["ssh", "%s" % HOST, COMMAND],
    shell=False,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE)
    result = ssh.stdout.readlines()
    if result == []:
        error = ssh.stderr.readlines()
        print(sys.stderr, "ERROR: %s" % error)
    else:
        print(result)



def t3():
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh_client.connect(hostname=hostname, username=mokgadi,password=mypassword, pkey=k, timeout=8)


    stdin,stdout,stderr=ssh_client.exec_command("ls")

    print(stdout.readlines())

    stdin, stdout, stderr = ssh.exec_command("sudo ls")
    stdin.write('mypassword\n')



# ssh = paramiko.SSHClient()
# ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
# ssh.connect('localhost',username='root',password='secret')
# chan = ssh.get_transport().open_session()
# chan.get_pty()
# chan.exec_command('tty')
# print(chan.recv(1024))



class SshRunner(Thread):
    def __init__(self, host, username, port=22, name=None, password=None, priv_key_file=None, timeout=8, print_stdout=False, print_stderr=True):
        Thread.__init__(self)
        self.host = host
        self.port = port
        self.name = name if name is not None else self.host
        self.username = username
        self.password = password
        self.priv_key_file = os.path.expanduser(priv_key_file.replace("$HOME", "~"))
        self.timeout  = timeout
        self.print_stdout = print_stdout
        self.print_stderr = print_stderr
        logger.info("{} {:<30s} pw ({} {})".format(self.name, "{}@{}:{}".format(self.username, self.host, self.port), self.password, self.priv_key_file, ))

        self.status = "idle"
        self.stop_flag = False
        self.cmd_queue = deque()

        self.ssh_client =paramiko.SSHClient()
        self.ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

        if self.priv_key_file:
            self.ssh_client.connect(hostname=self.host, port=port, username=username, key_filename=self.priv_key_file, timeout=timeout)
        else:
            self.ssh_client.connect(hostname=self.host, port=port, username=username, key_filename=self.priv_key_file, password=password, timeout=timeout)

        self.sftp_client = self.ssh_client.open_sftp()
        self.start()

    def run(self):
        # INFO("{:<8} {}@{:<30}:{} started".format(self.name, self.username, self.host, self.port))
        # logger.info("{} {}@{}:{} started".format(self.name, self.username, self.host, self.port))
        while (not self.stop_flag) or (len(self.cmd_queue) != 0):
            if len(self.cmd_queue) == 0:
                self.status = "idle"
                time.sleep(0.2)
            else:
                self.status = "running"
                cmd = self.cmd_queue.popleft()
                logger.debug("{} cmd {}".format(self.name, cmd))
                if isinstance(cmd, str):
                    self.run_cmd(cmd)
                elif isinstance(cmd, list) or isinstance(cmd, tuple):
                    if cmd[0] == "get":
                        self.run_get_file(cmd[1], cmd[2])
                    elif cmd[0] == "put":
                        self.run_put_file(cmd[1], cmd[2])
                    else:
                        raise RuntimeError("unknown command {}".format(cmd))
                else:
                    raise RuntimeError("unknown command {}".format(cmd))

        # INFO("{:<8} {}@{:<30}:{} stopped".format(self.name, self.username, self.host, self.port))
        logger.info("{} {}@{}:{} stopped".format(self.name, self.username, self.host, self.port))

    def run_cmd(self, cmd):
        stdin, stdout, stderr = self.ssh_client.exec_command(cmd)
        stdin, stdout, stderr = self.ssh_client.exec_command(cmd, get_pty=True)
        out = stdout.read().decode(errors="ignore").strip()
        err = stderr.read().decode(errors="ignore").strip()

        if self.print_stderr and len(err) != 0:
           # WARNING("{:<8} {}".format(self.name, err))
           logger.error("{} running cmd {} {}".format(self.name, cmd, err))

        if self.print_stdout and len(out):
            # DEBUG("{:<8} {}".format(self.name, out))
            logger.debug("{} {}".format(self.name, out))

    def add_cmd(self, cmd):
        self.cmd_queue.append(cmd)

    def add_cmds(self, cmds):
        for cmd in cmds:
            self.add_cmd(cmd)

    def put_file(self, local_file, remote_file):
        remote_file = remote_file.strip("/")
        if len(remote_file) == 0 or remote_file == "." or remote_file == "~" or remote_file == "$HOME":
            remote_file = local_file
        elif "." not in remote_file.split("/")[-1]:
            remote_file += "/" + local_file.split("/")[-1]
        self.cmd_queue.append(("put", local_file, remote_file))

    def get_file(self, remote_file, local_file=None):
        if local_file == None:
            local_file = remote_file.split("/")[-1]
        self.cmd_queue.append(("get", remote_file, local_file))


    def run_put_file(self, local_file, remote_file):
        # self.sftp_client = self.ssh_client.open_sftp()
        # remote file must have a name
        assert(remote_file[-1] != "/")
        self.sftp_client.put(os.path.expanduser(local_file), remote_file)


    def run_get_file(self, remote_file, local_file=None):
        # self.sftp_client = self.ssh_client.open_sftp()
        self.sftp_client.get(remote_file, os.path.expanduser(local_file))

    def stop(self):
        self.wait_to_stop()

    def wait_to_stop(self):
        while self.status == "running":
            time.sleep(0.2)

        self.stop_flag = True
        while len(self.cmd_queue) != 0:
            assert self.status != "idle"
            time.sleep(2)
        self.sftp_client.close()
        self.ssh_client.close()



class SshRunnerGroup:
    """ all host in the SshRunnerGroup runs the same command

    """
    def __init__(self, hosts, username, ports=[], names=[], password=None, priv_key_file=None, timeout=8, print_stdout=True, print_stderr=True):
        self.ssh_runners = {}

        if len(names) == 0:
            names = hosts
        assert len(hosts) == len(names)
        if len(ports) < len(hosts):
            for i in range(len(hosts) - len(ports)):
                ports.append(22)

        for i in range(len(hosts)):
            host, name, port = hosts[i], names[i], ports[i]
            ssh_runner = SshRunner(host, username, port, name, password, os.path.expanduser(priv_key_file.replace("$HOME", "~")),
                timeout=timeout, print_stdout=print_stdout, print_stderr=print_stderr)
            ssh_runner.start()
            self.ssh_runners[name] = ssh_runner


    def add_cmd(self, cmd):
        for ssh_runner in self.ssh_runners.values():
            ssh_runner.add_cmd(cmd)


    def add_cmds(self, cmds):
        for ssh_runner in self.ssh_runners.values():
            ssh_runner.add_cmds(cmds)

    def get_file(self, remote_file, local_file):
        for i, ssh_runner in enumerate(self.ssh_runners.values()):
            if local_file == None:
                local_file = remote_file.split("/")[-1] + "." + self.names[i]

            ssh_runner.get_file(remote_file, local_file)

    def put_file(self, local_file, remote_file):
        for ssh_runner in self.ssh_runners.values():
            ssh_runner.put_file(local_file, remote_file)


    def stop(self):
        for ssh_runner in self.ssh_runners.values():
            ssh_runner.stop()


def test_case1():
    sr = SshRunner("asrock.jasony.me", "jason", name="asrock", priv_key_file=os.path.expanduser("~/.ssh/id_rsa"))
    sr.start()
    sr.add_cmd("lsl")
    sr.put_file("sshRunner.py", "")
    time.sleep(2)
    sr.stop()


def test_case2():
    runner_group = SshRunnerGroup(["asrock.jasony.me", "d30.jasony.me", "mjolnir.mathcs.emory.edu"], "jason", names=["asrock", "d30", "mjolnir"], ports=[22, 22, 8098], priv_key_file=os.path.expanduser("~/.ssh/id_rsa"))
    runner_group.add_cmds(["hostname", "mkdir t; cd t; pwd; cd ..; rmdir t;"])
    time.sleep(2)
    runner_group.stop()

if __name__ == "__main__":
    test_case1()

