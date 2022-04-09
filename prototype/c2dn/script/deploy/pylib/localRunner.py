# -*- coding: utf-8 -*-
from __future__ import unicode_literals


import os, sys, time, subprocess, string
from collections import deque
from threading import Thread
from utils.printing import *
from utils.mylogging import get_logger

logger = get_logger(__name__)


class LocalRunner(Thread):
    def __init__(self, name, print_stdout=False):
        Thread.__init__(self)
        self.name = name
        self.print_stdout = print_stdout
        self.stop_flag = False
        self.cmd_list = deque()
        self.cmd_history = []
        self.status = "initialized"
        self.start()


    def add_cmd(self, cmd):
        self.cmd_list.append(cmd)

    def run(self):
        while (not self.stop_flag) or len(self.cmd_list) != 0:
            if len(self.cmd_list) == 0:
                time.sleep(0.2)
                continue
            cmd = self.cmd_list.popleft()
            logger.info(cmd)
            try:
                self.status = "running {}".format(cmd)
                p = subprocess.run(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True, text=True)
            except subprocess.CalledProcessError as e:
                logger.warning(str(e))
                logger.warning(p.stderr)
            else:
                self.cmd_history.append(cmd)
                if self.print_stdout:
                    print(p.stdout.strip(string.whitespace))
            finally:
                self.status = "idle"

        logger.info("localRunner {} finished".format(self.name))

    def wait(self):
        wait_times = 0
        while self.status.startswith("running") or len(self.cmd_list) > 0:
            time.sleep(1)
            wait_times += 1
            if wait_times % 20 == 0:
                logger.info("waiting for {}, {} {} cmds left".format(self.name, self.status, len(self.cmd_list)))



    def stop(self):
        time.sleep(2)
        self.stop_flag = True


if __name__ == "__main__":
    lr = LocalRunner("localrunner1")
    lr.start()
    lr.add_cmd("ls")
    lr.add_cmd("lsl")
    lr.stop()
