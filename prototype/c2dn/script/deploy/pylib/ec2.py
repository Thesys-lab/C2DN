# -*- coding: utf-8 -*-
from __future__ import unicode_literals
import boto3
import yaml
from pprint import pprint
import base64

import os, sys
from collections import defaultdict
sys.path.append(os.path.dirname(__file__))
from utils.printing import *
from config import load_confifg

# ec2 tutorial https://gist.github.com/nguyendv/8cfd92fc8ed32ebb78e366f44c2daea6

# there should a config.yaml or config.secret.yaml under working directory
# self.myawsconfig = load_confifg()


class EC2:
    def __init__(self, config_path="config.secret.yaml"):
        self.config_path = config_path
        self.myawsconfig = load_confifg(self.config_path)
        INFO("config {} AWS EC2 use region {}".format(config_path, self.myawsconfig["aws"]["region"]))
        self.ec2client = boto3.client("ec2", region_name=self.myawsconfig["aws"]["region"])
        self.ec2resource = boto3.resource('ec2', region_name=self.myawsconfig["aws"]["region"])

    def get_instances_info(self, states=("running", ), keyname=("Jason", ), **kwargs):
        filters = [
            {"Name": "instance-state-name", "Values": states},
            {"Name": "key-name", "Values": keyname},
        ]
        if len(kwargs) == 0:
            WARNING("no filter")

        if "ami_names" in kwargs:
            ami_ids = list(self.myawsconfig["ami_id"][ami_name] for ami_name in kwargs["ami_names"])
            filters.append({"Name": "image-id", "Values": ami_ids})
        elif "ami_ids" in kwargs:
            ami_ids = kwargs["ami_ids"]
            filters.append({"Name": "image-id", "Values": ami_ids})

        if "names" in kwargs:
            instance_names = kwargs["names"]
            filters.append({"Name": "tag:Name", "Values": instance_names})


        if "launch_time" in kwargs:
            launch_time = kwargs["launch_time"]

        if "instance_ids" in kwargs:
            instance_ids = kwargs.get("instance_ids")
            filters.append({"Name": "instance-id", "Values": instance_ids})

        if "instance_ips" in kwargs:
            instance_ips = kwargs.get("instance_ips")
            filters.append({"Name": "ip-address", "Values": instance_ips})

        assert "instance_id" not in kwargs and "instance_ip" not in kwargs, "please use instance_ips and instance_ids"


        # DEBUG(filters)
        resp = self.ec2client.describe_instances(Filters=filters)


        if resp["ResponseMetadata"]["HTTPStatusCode"] != 200:
            ERROR(resp)
            raise RuntimeError("retrieve instances info error")

        # pprint(resp)
        return resp

    def get_instances_id(self, **kwargs):
        resp = kwargs.get("resp", self.get_instances_info(**kwargs))

        ids = []
        for i in range(len(resp["Reservations"])):
            for j in range(len(resp["Reservations"][i]["Instances"])):
                ids.append(resp["Reservations"][i]["Instances"][j]["InstanceId"])
        return ids

    def get_public_ip(self, **kwargs):
        resp = kwargs.get("resp", self.get_instances_info(**kwargs))       # don't use this, either multiple instance request will always get first IP
        # resp = self.get_instances_info(**kwargs)
        ips = []
        for i in range(len(resp["Reservations"])):
            for j in range(len(resp["Reservations"][i]["Instances"])):
                ips.append(resp["Reservations"][i]["Instances"][j]["PublicIpAddress"])
        return ips

    def get_placement(self, **kwargs):
        resp = kwargs.get("resp", self.get_instances_info(**kwargs))
        placements = {}
        for i in range(len(resp["Reservations"])):
            for j in range(len(resp["Reservations"][i]["Instances"])):
                instance_id = resp["Reservations"][i]["Instances"][j]["InstanceId"]
                public_ip = resp["Reservations"][i]["Instances"][j]["PublicIpAddress"]
                private_ip = resp["Reservations"][i]["Instances"][j]["PrivateIpAddress"]
                placement = resp["Reservations"][i]["Instances"][j]["Placement"]

                placements[instance_id] = (public_ip, private_ip, placement)
                # placements[instance_id] = placement

        return placements

    def get_private_ip(self, **kwargs):
        resp = kwargs.get("resp", self.get_instances_info(**kwargs))
        ips = []
        for i in range(len(resp["Reservations"])):
            for j in range(len(resp["Reservations"][i]["Instances"])):
                ips.append(resp["Reservations"][i]["Instances"][j]["PrivateIpAddress"])
        return ips

    def create_spot_instances(self, instance_type, ami_id=None, ami_name=None, n_instance=1, names=("spot", ), wait_for_instance_init=True, **kwargs):
        assert ami_id is not None or ami_name is not None, "please provide either ami_id or ami_name"
        if ami_id is None:
            ami_id = self.myawsconfig["ami_id"][ami_name]
        assert len(names) == 1 or len(names) == n_instance, "names should be a list/tuple of length 1 or n_instance"
        if len(names) == 1:
            name = names[0]
            names = ["{}_{}".format(name, i) for i in range(n_instance)]
        spot_request_ids, instance_ids, instance_ips = [], [], []

        userdata = kwargs.pop("userdata", "")
        userdata = base64.b64encode(userdata.encode()).decode()

        launch_spec = {
                # 'SecurityGroupIds': ['sg-0c1e392ca2853d492', ],
                'SecurityGroups': ['default', ],
                # 'EbsOptimized': True,
                'ImageId': ami_id,
                'InstanceType': instance_type,
                'KeyName': 'Jason',
                'UserData': userdata,
            }

        if "availability_zone" in kwargs:
            launch_spec["Placement"] = {'AvailabilityZone': kwargs.pop("availability_zone")}
        if "placement_group" in kwargs and kwargs["placement_group"]:
            launch_spec["Placement"] = {'GroupName': kwargs.pop("placement_group")}
        if "disk" in kwargs:
            launch_spec["BlockDeviceMappings"] = kwargs.pop("disk")
        if "ebs_opt" in kwargs:
            launch_spec["EbsOptimized"] = kwargs.pop("ebs_opt", False)


        launch_spec = kwargs.pop("launch_spec", launch_spec)
        assert len(kwargs) == 0, print("left over kwargs {}".format(kwargs))

        resp = self.ec2client.request_spot_instances(
            InstanceCount=n_instance,
            LaunchSpecification=launch_spec,
            SpotPrice="200",
            **kwargs)

        # pprint(resp)
        if resp["ResponseMetadata"]["HTTPStatusCode"] != 200:
            pprint(resp)
            raise RuntimeError("unknown error during spot instance requests")

        for i in range(len(resp['SpotInstanceRequests'])):
            spot_request_ids.append(resp['SpotInstanceRequests'][i]['SpotInstanceRequestId'])


        INFO("Wait for request to be fulfilled...")
        waiter = self.ec2client.get_waiter('spot_instance_request_fulfilled')
        for i, request_id in enumerate(spot_request_ids):
            waiter.wait(SpotInstanceRequestIds=[request_id])
            INFO("{} requests fulfilled".format(i+1))


        resp = self.ec2client.describe_spot_instance_requests(SpotInstanceRequestIds=spot_request_ids)


        for i in range(n_instance):
            if resp["SpotInstanceRequests"][i]["Status"]["Code"] != "fulfilled":
                raise RuntimeError(resp["SpotInstanceRequests"][i]["Status"]["Message"])
            instance_ids.append(resp["SpotInstanceRequests"][i]["InstanceId"])

        # wait for instance to be ready
        waiter = self.ec2client.get_waiter('instance_running')
        waiter.wait(Filters=[{"Name": "instance-id", "Values": instance_ids}])

        for instance_id in instance_ids:
            ips = self.get_public_ip(instance_ids=(instance_id,))
            instance_ips.append(ips[0])

        for instance_id, name in zip(instance_ids, names):
            self.ec2client.create_tags(Resources=(instance_id, ), Tags=[{"Key": "Name", "Value": name}])


        if wait_for_instance_init:
            INFO("wait for instances to be ready")
            # waiter = self.ec2client.get_waiter('instance_running')
            # waiter.wait(Filters=[{"Name": "instance-id", "Values": instance_ids}])
            waiter = self.ec2client.get_waiter('instance_status_ok')
            waiter.wait(InstanceIds=instance_ids)

        return instance_ips, instance_ids

    def create_instances(self, instance_type, ami_id=None, ami_name=None, n_instance=1, names=("ondemand", ), wait_for_instance_init=True, **kwargs):
        assert ami_id is not None or ami_name is not None, "please provide either ami_id or ami_name"
        if ami_id is None:
            ami_id = self.myawsconfig["ami_id"][ami_name]

        assert len(names) == 1 or len(names) == n_instance, "names should be a list/tuple of length 1 or n_instance"
        if len(names) == 1:
            name = names[0]
            names = ["{}_{}".format(name, i) for i in range(n_instance)]
        instance_ids, instance_ips = [], []

        userdata = kwargs.pop("userdata", "")
        userdata = base64.b64encode(userdata.encode()).decode()

        launch_spec = kwargs.pop("launch_spec", defaultdict(dict))
        if "availability_zone" in kwargs:
            launch_spec["Placement"]['AvailabilityZone'] = kwargs.pop("availability_zone")
        if "placement_group" in kwargs and kwargs["placement_group"]:
            launch_spec["Placement"]['GroupName'] = kwargs.pop("placement_group")
        if "disk" in kwargs:
            launch_spec["BlockDeviceMappings"] = kwargs.pop("disk")
        if "ebs_opt" in kwargs:
            launch_spec["EbsOptimized"] = kwargs.pop("ebs_opt", False)

        assert(len(kwargs) == 0), "kwargs is {}".format(kwargs)


        print("************** USER_DATA seems not working **************")

        # pprint(launch_spec)
        resp = self.ec2resource.create_instances(
            MinCount=n_instance,
            MaxCount=n_instance,
            InstanceType=instance_type,
            ImageId=ami_id,
            KeyName="Jason",
            SecurityGroups=["default", ],
            UserData=userdata,
            **launch_spec)

        for ins in resp:
            instance_ids.append(ins.id)

        # wait for instance to be ready
        waiter = self.ec2client.get_waiter('instance_running')
        waiter.wait(Filters=[{"Name": "instance-id", "Values": instance_ids}])

        for instance_id in instance_ids:
            ips = self.get_public_ip(instance_ids=(instance_id,))
            instance_ips.append(ips[0])

        for instance_id, name in zip(instance_ids, names):
            self.ec2client.create_tags(Resources=(instance_id, ), Tags=[{"Key": "Name", "Value": name}])


        if wait_for_instance_init:
            INFO("wait for instances to be ready")
            # waiter = self.ec2client.get_waiter('instance_running')
            # waiter.wait(Filters=[{"Name": "instance-id", "Values": instance_ids}])
            waiter = self.ec2client.get_waiter('instance_status_ok')
            waiter.wait(InstanceIds=instance_ids)

        return instance_ips, instance_ids

    def stop_instances(self, ids):
        response = self.ec2client.stop_instances(InstanceIds=ids, )

    def terminate_instances(self, ids):
        self.ec2client.terminate_instances(InstanceIds=ids)

    def update_instance_name(self, instance_id, name):
        self.ec2client.create_tags(Resources=[instance_id, ], Tags=[{'Key': 'Name', 'Value': name}])


if __name__ == "__main__":
    # pprint(get_public_ip(ami_names="ats"))
    pprint(EC2().get_placement(ami_names=("ats", )))
    pprint(EC2().get_placement(instance_ids=("i-066fd3492ba442af8", )))
    # create_spot_instances(n=2, instance_type="t2.small", name="jasonTest")
    # ids = EC2().get_instances_id(names=["jasonTest2", ])
    # pprint(ids)
    # EC2().terminate_instances(ids)
    # EC2().create_spot_instances(ami_id="ami-0ca70ab9de534d4a6", n=2, instance_type="t2.small", names=("jasonTest", "test2"), userdata=s)
    # ips, ids = EC2().create_instances(ami_id="ami-0ca70ab9de534d4a6", instance_type="t2.small", n=2, userdata=s)
    # EC2().stop_instances(ids)


