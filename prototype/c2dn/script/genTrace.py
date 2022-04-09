"""
this script scales down the original trace 

""" 


import os, sys
import struct 


def gen_trace(ifilepath, ofilepath, scale_down_ratio=10):
    s = struct.Struct("<III")
    new_trace_obj_dict = {}
    n_req = 0

    with open(ifilepath, "rb") as ifile:
        with open(ofilepath, "wb") as ofile:
            data = ifile.read(s.size)
            start_ts = s.unpack(data)[0]
            while data:
                ts, obj, sz = s.unpack(data) 
                if obj % 10001 > 10001//scale_down_ratio:
                    data = ifile.read(s.size)
                    continue
                ts = (ts - start_ts) // scale_down_ratio
                ofile.write(s.pack(ts, obj, sz)) 
                n_req += 1
                new_trace_obj_dict[obj] = sz
                data = ifile.read(s.size)

    print("new trace has {} req {} obj {:.2f}GB".format(n_req, len(new_trace_obj_dict), sum(new_trace_obj_dict.values())/1024/1024/1024))


if __name__ == "__main__":
    gen_trace("/disk1/CDN/akamai/web/akamai.bin", "/disk1/CDN/akamai/web/akamai.bin.scale10")
    # gen_trace("/disk1/CDN/akamai/video/akamai2.bin", "/disk1/CDN/akamai/video/akamai2.bin.scale10") 

