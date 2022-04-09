# -*- coding: utf-8 -*-
from __future__ import unicode_literals

import yaml

import os, sys
sys.path.append(os.path.dirname(__file__))
from utils.printing import *


def load_confifg(config_path="config.yaml"):
    if os.path.exists("config.secret.yaml"):
        config_path = "config.secret.yaml"
        INFO("load secret config")
    # else:
    #     pprint(os.listdir("."))

    with open(config_path) as ifile:
        config = yaml.load(ifile)

    return config


if __name__ == "__main__":
    pprint(load_confifg())
