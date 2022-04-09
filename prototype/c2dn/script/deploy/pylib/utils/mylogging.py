# -*- coding: utf-8 -*-
from __future__ import unicode_literals


import logging


def get_logger(name, filename='jobSubmit.log', logging_level=logging.DEBUG):
    logger = logging.getLogger(name)
    logger.setLevel(logging_level)

    handler = logging.FileHandler(filename=filename, mode='w', )
    handler.setLevel(logging.DEBUG)
    handler.setFormatter(logging.Formatter('%(asctime)s [%(levelname)-5s] %(name)+16s - [%(threadName)s]: %(message)s '))
    logger.addHandler(handler)

    handler2 = logging.StreamHandler()
    handler2.setLevel(logging.INFO)
    handler2.setFormatter(logging.Formatter('%(asctime)s [%(levelname)-5s] %(name)+16s - [%(threadName)s]: %(message)s '))
    logger.addHandler(handler2)


    return logger
