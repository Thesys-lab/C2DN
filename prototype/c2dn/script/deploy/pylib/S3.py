# -*- coding: utf-8 -*-
from __future__ import unicode_literals


import os
import boto3
from utils.printing import *

class S3:
    client = boto3.client("s3")
    def upload_file(filepath, bucket):
        filepath = os.path.normpath(filepath)
        DEBUG("upload {} to {}".format(filepath, bucket))
        S3.client.upload_file(filepath, bucket, filepath)

    def upload_dir(dirname, bucket):
        for f in os.listdir(dirname):
            if f.startswith("_") or f.startswith("."):
                continue

            if os.path.isfile(dirname + "/" + f):
                S3.upload_file(dirname + "/" + f, bucket)
            else:
                S3.upload_dir(dirname + "/" + f, bucket)


if __name__ == "__main__":
    S3.upload_dir(".", "jasontest2018")

