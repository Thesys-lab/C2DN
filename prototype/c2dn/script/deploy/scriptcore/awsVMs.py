#!/usr/bin/env python3
# -*- coding: utf-8 -*-
from __future__ import unicode_literals


import os, sys, time, glob, subprocess, logging, math
import yaml
from threading import Thread
from pprint import pprint
from copy import copy

sys.path.append("../")

from pylib.ec2 import *
# from pylib.sshRunner import SshRunner
# from pylib.localRunner import LocalRunner
import pylib.utils.printing as printing
from pylib.utils.printing import *
from pylib.utils.mylogging import get_logger
from pylib.config import load_confifg


logger = get_logger(__name__)
printing.print_level = PRINT_LEVEL_DEBUG
CONFIG_DIR = "config/"
CONFIG_DIR = os.path.normpath("{}/awsconfig/".format(os.path.dirname(os.path.abspath(__file__))))


class Client:
    def __init__(self, config_path, n_instance=1, ami_name="ubuntu18", instance_type="c5n.4xlarge", spot=True, **kwargs):
        self.n_instance = n_instance
        self.ami_name = ami_name
        self.config_path = config_path
        self.myawsconfig = load_confifg(config_path)
        self.ec2 = EC2(config_path=config_path)
        self.n_instance = n_instance
        self.spot = spot
        self.instance_type = instance_type
        self.names = kwargs.pop("names", ["client", ])
        self.kwargs = kwargs
        self.userdata = ""
        self.public_ips, self.private_ips = [], []
        self.disk = [
        {'DeviceName': '/dev/sda1', # 'VirtualName': 'string', 'NoDevice': 'string'
         'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 80, 'VolumeType': 'gp2'},}]
        # {'DeviceName': '/dev/sda1', # 'VirtualName': 'string', 'NoDevice': 'string'
        #  'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 800, 'VolumeType': 'gp2', 'Encrypted': False, },}]
                # 'Iops': 123, 'SnapshotId': 'string', 'KmsKeyId': 'string'
        self.placement_group = kwargs.pop("placement_group", None)
        assert len(kwargs) == 0, str(kwargs)


    def launch_VM(self):
        if self.spot:
            func = self.ec2.create_spot_instances
        else:
            func = self.ec2.create_instances

        _, self.instance_ids = func(ami_name=self.ami_name, n_instance=self.n_instance,
            instance_type=self.instance_type, names=self.names, userdata=self.userdata, disk=self.disk, **self.kwargs)

        placements = self.ec2.get_placement(instance_ids=self.instance_ids)
        for instance in placements.values():
            INFO(instance)
            self.public_ips.append(instance[0])
            self.private_ips.append(instance[1])

        assert len(self.public_ips) == len(self.instance_ids)
        INFO("client public_ips {}, private_ips".format(self.public_ips, self.private_ips))


    def get_ips(self):
        return self.public_ips, self.private_ips


class Origin:
    def __init__(self, config_path, n_instance=1, ami_name="ats", instance_type="c5.2xlarge", spot=True, **kwargs):
        self.n_instance = n_instance
        self.ami_name = ami_name
        self.config_path = config_path
        self.myawsconfig = load_confifg(config_path)
        self.ec2 = EC2(config_path=config_path)
        self.n_instance = n_instance
        self.spot = spot
        self.instance_type = instance_type
        self.names = kwargs.pop("names", ["origin", ])
        self.kwargs = kwargs
        self.userdata = ""
        self.public_ips, self.private_ips = [], []


    def launch_VM(self):
        if self.spot:
            func = self.ec2.create_spot_instances
        else:
            func = self.ec2.create_instances
        _, self.instance_ids = func(ami_name=self.ami_name, n_instance=self.n_instance,
                instance_type=self.instance_type, names=self.names, userdata=self.userdata, **self.kwargs)

        placements = self.ec2.get_placement(instance_ids=self.instance_ids)
        for instance in placements.values():
            INFO(instance)
            self.public_ips.append(instance[0])
            self.private_ips.append(instance[1])

        assert len(self.public_ips) == len(self.instance_ids)
        INFO("origin public_ips {}, private_ips {}".format(self.public_ips, self.private_ips))


    def get_ips(self):
        return self.public_ips, self.private_ips


class CDN:
    def __init__(self, config_path, n_instance, ami_name="ats", instance_type="m5d.4xlarge", disk=(), placement_group="ats", spot=True, **kwargs):
        self.config_path = config_path
        self.myawsconfig = load_confifg(config_path)
        self.ec2 = EC2(config_path=config_path)
        self.n_instance = n_instance
        self.spot = spot
        self.ami_name = ami_name
        self.instance_type = instance_type
        # self.placement_group = placement_group
        self.names = kwargs.pop("names", ["CDN", ])
        self.disk = disk
        if len(self.disk) == 0:
            self.disk = [
            {'DeviceName': '/dev/sda1', # 'VirtualName': 'string', 'NoDevice': 'string'
             'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 80, 'VolumeType': 'gp2'},}]
        self.kwargs = kwargs
        # if len(self.disk) == 0:
        #     self.disk = [
        #     {'DeviceName': '/dev/sdb', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        #             # 'Iops': 123, 'SnapshotId': 'string', 'KmsKeyId': 'string'
        #     {'DeviceName': '/dev/sdc', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        #     {'DeviceName': '/dev/sdd', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        #     {'DeviceName': '/dev/sde', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        #     {'DeviceName': '/dev/sdf', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        #     {'DeviceName': '/dev/sdg', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        #     {'DeviceName': '/dev/sdh', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        #     {'DeviceName': '/dev/sdi', # 'VirtualName': 'string', 'NoDevice': 'string'
        #      'Ebs': {'DeleteOnTermination': True, 'VolumeSize': 600, 'VolumeType': 'standard', 'Encrypted': False, },},
        # ]


        self.userdata = ""
        self.instance_ids, self.public_ips, self.private_ips = [], [], []
        # self.sshrunners = []


    def launch_VM(self):
        if self.spot:
            func = self.ec2.create_spot_instances
        else:
            func = self.ec2.create_instances
        # placement_group=self.placement_group,
        # kw = {}
        # if self.placement_group is not None:
        #     kw["placement_group"] = self.placement_group
        # if
        _, self.instance_ids = func(ami_name=self.ami_name, n_instance=self.n_instance, instance_type=self.instance_type, names=self.names,
            userdata=self.userdata, disk=self.disk, ebs_opt=True, **self.kwargs)

        # placements = self.ec2.get_placement(instance_ids=self.instance_ids)
        # for instance in placements.values():
        #     INFO(instance)
        #     self.public_ips.append(instance[0])
        #     self.private_ips.append(instance[1])

        for i in range(0, self.n_instance):
            self.public_ips.append(self.ec2.get_public_ip(names=["{}_{}".format(self.names[0], i), ])[0])
            self.private_ips.append(self.ec2.get_private_ip(names=["{}_{}".format(self.names[0], i), ])[0])
        # print("export cdn_pub_ips=({})".format(" ".join(self.public_ips)))
        # print("export cdn_priv_ips=({})".format(" ".join(self.private_ips)))



        # for ip in self.public_ips:
        #     self.sshrunners.append(SshRunner(ip, "ubuntu"))

        # INFO("public_ips {}".format(self.public_ips))
        # INFO("private_ips {}".format(self.private_ips))


    def get_ips(self):
        return self.public_ips, self.private_ips


def test(ami_id="ami-0f3a518006c14b17a", instance_type="c5d.2xlarge", n=1, placement_group="ats", userdata=""):
    # vpc="vpc-0bab30f8e17919792", public_ip=True,

    userdata = base64.b64encode(userdata.encode()).decode()
    print(base64.b64decode(userdata.encode()))
    # return

    launch_spec = {
            'SecurityGroups': ['Jason', ],
            'ImageId': ami_id,
            'InstanceType': instance_type,
            'KeyName': 'Jason',
            'UserData': userdata,
            "Placement": {'GroupName': placement_group}
        }


    instance_ips, instance_ids = EC2.create_spot_instances(None, None, launch_spec=launch_spec)
    print(instance_ids, instance_ips)

    placements = EC2.get_placement(instance_ids=instance_ids)
    print(placements)


def test2(ami_name="ats", ami_id=None, instance_type="c5.2xlarge", n=1):
    # config_path = "config.secret.eu.west2.yaml"
    config_path = "config.secret.ap.northeast1.yaml"
    myawsconfig = load_confifg(config_path)
    ami_id = myawsconfig["ami_id"][ami_name]
    print(ami_id)
    # ec2 = EC2(config_path="config.secret.ap.southeast2.yaml")

    ec2 = EC2(config_path=config_path)
    instance_ips, instance_ids = ec2.create_spot_instances(ami_id, instance_type, n=n)
    print(instance_ids, instance_ips)

    placements = ec2.get_placement(instance_ids=instance_ids)
    print(placements)


def test3():
    cdn = CDN(n_instance=10, ami_name="ubuntu18", instance_type="t3.large", names=["test", ], spot=False,
                config_path=CONFIG_DIR+"/config.secret.us-east-1.yaml")
    cdn.launch_VM()
    cdn_public_ips, cdn_private_ips = cdn.get_ips()
    with open("/tmp/cdnPub", "w") as ofile:
        for ip in cdn_public_ips:
            ofile.write("{}\n".format(ip))
    with open("/tmp/cdnPrv", "w") as ofile:
        for ip in cdn_private_ips:
            ofile.write("{}\n".format(ip))
    print("export cdn_pub_ips=({})".format(" ".join(cdn_public_ips)))
    print("export cdn_priv_ips=({})".format(" ".join(cdn_private_ips)))


def getCDNips(name):
    for idx in range(1, 3):
        try:
            ec2 = EC2(config_path=CONFIG_DIR+"/config.secret.us-east-{}.yaml".format(idx))
            public_ips = []
            private_ips = []
            # if not name.startswith("CDN_"):
            name = "CDN_" + name
            # print("using east-{}, get ip for {}".format(idx, name))
            for i in range(0, 10):
                public_ips.append(ec2.get_public_ip(names=["{}_{}".format(name, i), ])[0])
                private_ips.append(ec2.get_private_ip(names=["{}_{}".format(name, i), ])[0])
            print("export cdn_pub_ips=({})".format(" ".join(public_ips)))
            print("export cdn_priv_ips=({})".format(" ".join(private_ips)))
            return public_ips, private_ips
        except:
            pass

def terminateVMs(availability_zone, name=None, ip=None):
    print("terminateVMs name {} ip {}".format(name, ip))
    ec2 = EC2(config_path=CONFIG_DIR+"/config.secret.{}.yaml".format(availability_zone[:-1]))

    if ip is None:
        ip = getCDNips(name)[0]

    ids = ec2.get_instances_id(instance_ips=ip)
    print(ids)
    ec2.terminate_instances(ids)


def handle_cmd(arg):
    if arg[1] == "CDN":
        assert len(arg) > 4
        names = ("{}_{}".format(arg[1], arg[2]), )
        placement_group = arg[3]
        availability_zone = arg[4]

        # cdn = CDN(n_instance=10, ami_name="ubuntu18", instance_type="m5d.4xlarge", names=names,
        # cdn = CDN(n_instance=10, ami_name="ubuntu18", instance_type="t3.large", names=names,
        # cdn = CDN(n_instance=10, ami_name="ubuntu18", instance_type="i3.8xlarge", names=names,
        cdn = CDN(n_instance=10, ami_name="ubuntu18", instance_type="i3en.6xlarge", names=names, 
                    config_path=CONFIG_DIR+"/config.secret.{}.yaml".format(availability_zone[:-1]),
                    placement_group=placement_group, spot=spot,
                    availability_zone=availability_zone)
        cdn.launch_VM()
        cdn_public_ips, cdn_private_ips = cdn.get_ips()
        with open("/tmp/cdnPub", "w") as ofile:
            for ip in cdn_public_ips:
                ofile.write("{}\n".format(ip))
        with open("/tmp/cdnPrv", "w") as ofile:
            for ip in cdn_private_ips:
                ofile.write("{}\n".format(ip))
        print("export cdn_pub_ips=({})".format(" ".join(cdn_public_ips)))
        print("export cdn_priv_ips=({})".format(" ".join(cdn_private_ips)))
        with open("/tmp/clusterIP", "a") as ofile:
            ofile.write("export cdn_pub_ips=({})".format(" ".join(cdn_public_ips)))
            ofile.write("\n")
            ofile.write("export cdn_priv_ips=({})".format(" ".join(cdn_private_ips)))
            ofile.write("\n")
        # with open("/tmp/cdnhost", "w") as ofile:
        #     for ip in cdn_public_ips:
        #         ofile.write(f"{ip}\n")
        

    elif arg[1] == "origin":
        names = ("{}_{}".format(arg[1], arg[2]), )
        # origin = Origin(config_path=CONFIG_DIR+"/config.secret.ap.northeast1.yaml", names=names,
        origin = Origin(config_path=CONFIG_DIR+"/config.secret.eu-central-1.yaml", ami_name="ubuntu18", names=names,
                           # instance_type="m5.xlarge", spot=spot)
                           instance_type="t3a.2xlarge", spot=spot)
        origin.launch_VM()
        origin_public_ips, origin_private_ips = origin.get_ips()
        print("export origin_ip=" + " ".join(origin_public_ips))
        with open("/tmp/origin", "w") as ofile:
            for ip in origin_public_ips:
                ofile.write("{}".format(ip))
        with open("/tmp/clusterIP", "a") as ofile:
            ofile.write("export origin_ip=" + " ".join(origin_public_ips))
            ofile.write("\n")


    elif arg[1] == "client":
        names = ("{}_{}".format(arg[1], arg[2]), )
        client = Client(config_path=CONFIG_DIR+"/config.secret.ca-central-1.yaml", ami_name="ubuntu18", names=names,
                            # instance_type="m5.xlarge", spot=spot)
                            instance_type="t3a.2xlarge", spot=spot)
        client.launch_VM()
        client_public_ips, client_private_ips = client.get_ips()
        print("export client_ip=" + " ".join(client_public_ips))
        with open("/tmp/client", "w") as ofile:
            for ip in client_public_ips:
                ofile.write("{}".format(ip))
        with open("/tmp/clusterIP", "a") as ofile:
            ofile.write("export client_ip=" + " ".join(client_public_ips))
            ofile.write("\n")

    elif arg[1] == "getCDNips":
        getCDNips(arg[2])

    elif arg[1] == "terminateVMsIP":
        terminateVMs(availability_zone=arg[2], ip=arg[3:])

    elif arg[1] == "terminateVMsName":
        terminateVMs(availability_zone=arg[2], name=arg[3], )

    elif arg[1] == "test3":
        test3()

    else:
        print(arg)
        raise RuntimeError("unknown role " + arg[1])


if __name__ == "__main__":
    spot=True
    if len(sys.argv) < 2:
        print("usage: {} func/VMType[origin/CDN/client] name placementGroup".format(sys.argv[0]))
        sys.exit(1)
    handle_cmd(sys.argv)

