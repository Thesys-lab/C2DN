

import os, sys
sys.path.append(os.path.expanduser("~/workspace/"))
from pyutils.common import * 
import functools 


bucket_to_server_map = {0:[8,9,4,0], 1:[0,2,4,9], 2:[0,3,1,2], 3:[3,7,0,1], 4:[7,3,1,0], 5:[0,4,5,1], 6:[0,1,4,2], 7:[6,0,5,1], 8:[2,7,9,0], 9:[9,3,0,1], 10:[6,3,9,0], 11:[6,9,4,0], 12:[5,7,3,0], 13:[2,7,9,0], 14:[7,3,1,0], 15:[3,7,0,1], 16:[0,6,9,1], 17:[2,4,5,0], 18:[5,0,4,1], 19:[3,4,2,0], 20:[5,9,4,0], 21:[9,2,3,0], 22:[3,4,8,0], 23:[9,8,3,0], 24:[7,9,1,0], 25:[7,2,5,0], 26:[7,6,9,0], 27:[2,7,9,0], 28:[7,9,5,0], 29:[2,0,1,3], 30:[0,6,5,1], 31:[5,1,7,0], 32:[1,0,7,2], 33:[4,8,0,1], 34:[0,9,4,1], 35:[4,6,8,1], 36:[1,6,2,3], 37:[4,6,7,1], 38:[1,9,3,2], 39:[9,4,3,1], 40:[7,8,5,1], 41:[9,6,0,1], 42:[5,4,7,1], 43:[4,5,9,1], 44:[6,4,3,1], 45:[2,9,8,1], 46:[9,2,7,1], 47:[1,6,2,3], 48:[4,8,2,1], 49:[8,0,4,1], 50:[6,9,7,1], 51:[6,9,7,1], 52:[0,7,1,2], 53:[2,5,3,1], 54:[4,9,7,1], 55:[0,9,4,1], 56:[1,2,9,3], 57:[8,3,2,1], 58:[5,3,9,2], 59:[2,0,3,4], 60:[4,3,2,5], 61:[3,8,5,2], 62:[5,2,3,4], 63:[2,6,4,3], 64:[0,1,4,2], 65:[8,9,4,2], 66:[4,6,0,2], 67:[9,4,3,2], 68:[0,4,2,3], 69:[8,2,4,3], 70:[0,5,3,2], 71:[5,2,4,3], 72:[2,5,4,3], 73:[6,5,4,2], 74:[9,6,7,2], 75:[1,2,5,3], 76:[1,8,9,2], 77:[1,2,7,3], 78:[2,6,1,3], 79:[0,9,4,2], 80:[4,3,0,2], 81:[5,0,6,2], 82:[4,8,6,2], 83:[3,1,0,2], 84:[6,3,1,4], 85:[8,1,6,3], 86:[9,3,2,5], 87:[1,3,4,5], 88:[0,8,6,3], 89:[7,1,2,3], 90:[7,8,4,5], 91:[4,8,3,5], 92:[4,2,6,5], 93:[4,0,2,5], 94:[2,0,3,5], 95:[3,6,5,7], 96:[9,6,0,5], 97:[3,4,7,5], 98:[4,5,2,6], 99:[2,9,5,6], 100:[9,5,1,6], 101:[0,7,1,5], 102:[7,6,4,5], 103:[7,0,1,5], 104:[7,3,4,5], 105:[9,6,0,5], 106:[8,2,9,5], 107:[8,0,9,5], 108:[8,0,9,5], 109:[1,7,3,5], 110:[9,5,8,6], 111:[5,6,3,7], 112:[6,0,8,5], 113:[4,6,0,5], 114:[4,2,6,5], 115:[7,5,3,6], 116:[7,8,6,9], 117:[6,5,3,7], 118:[7,3,9,6], 119:[6,7,0,8], 120:[8,4,0,6], 121:[2,9,7,6], 122:[5,0,4,6], 123:[8,5,6,7], 124:[2,6,9,7], 125:[8,4,6,7], 126:[3,8,1,6], 127:[3,1,0,6], 128:[4,3,2,6], 129:[9,4,2,6], 130:[4,9,1,6], 131:[6,0,8,7], 132:[7,6,0,8], 133:[1,2,5,6], 134:[2,5,7,6], 135:[1,6,3,7], 136:[7,3,1,6], 137:[9,0,7,8], 138:[7,5,1,8], 139:[7,3,6,8], 140:[3,6,7,8], 141:[2,6,1,7], 142:[4,5,9,7], 143:[7,5,4,8], 144:[4,8,5,7], 145:[9,6,5,7], 146:[8,5,3,7], 147:[8,3,7,9], 148:[3,0,8,7], 149:[5,6,0,7], 150:[9,8,3,7], 151:[9,4,7,8], 152:[9,0,4,7], 153:[9,7,2,8], 154:[3,1,5,7], 155:[4,5,1,7], 156:[1,2,5,7], 157:[6,4,0,7], 158:[4,6,3,7], 159:[1,9,6,7], 160:[3,9,6,7], 161:[1,8,2,7], 162:[3,1,4,7], 163:[9,0,4,8], 164:[1,9,6,8], 165:[4,5,8,9], 166:[2,9,5,8], 167:[4,9,2,8], 168:[7,2,5,8], 169:[6,2,7,8], 170:[8,2,5,9], 171:[0,5,8,9], 172:[6,3,0,8], 173:[9,6,3,8], 174:[1,9,6,8], 175:[4,0,8,9], 176:[6,1,2,8], 177:[6,4,5,8], 178:[1,2,7,8], 179:[4,5,2,8], 180:[3,8,1,9], 181:[5,0,3,8], 182:[3,8,4,9], 183:[5,3,7,8], 184:[8,7,1,9], 185:[3,6,8,9], 186:[4,2,3,8], 187:[5,4,9,8], 188:[3,6,4,8], 189:[9,0,8,1], 190:[4,5,1,8], 191:[6,0,9,8], 192:[7,3,0,8], 193:[5,4,8,9], 194:[1,2,6,8], 195:[0,6,1,8], 196:[8,4,2,9], 197:[4,5,2,9], 198:[3,2,4,9], 199:[1,0,8,9], }
server_has_bucket_map = defaultdict(list)
for bucket, server_list in bucket_to_server_map.items():
    server_has_bucket_map[server_list[0]].append(bucket)
    server_has_bucket_map[server_list[1]].append(bucket)


@functools.lru_cache(16*1024*1024)
def load_data(ifilepath, system): 
    """ this can load latency of the merged files, and each latency point is not averaged, this is used for the CDF plot 
    to generate merged file, run: 
    cat client.latency.firstByte* > client.latency.firstByte.all; sort -n -k 1 client.latency.firstByte.all > client.latency.firstByte.all.sort

    """
    lat_large_obj, lat_small_obj, lat_all = [], [], []
    start_ts, final_ts = 0, 0

    with open(ifilepath) as ifile:
        for line in ifile:
            ts, req_str, latency = line.strip().split()
            ts, latency = int(ts), float(latency)
            obj, size = [int(f) for f in req_str.split("_")]
            large_obj = size > 128*1024 

            if start_ts == 0:
                start_ts = ts
                last_ts = ts
            final_ts = ts

            lat_all.append(latency)
            if large_obj:
                lat_large_obj.append(latency)
            else:
                lat_small_obj.append(latency)

    return final_ts-start_ts, lat_large_obj, lat_small_obj, lat_all


@functools.lru_cache(16*1024*1024)
def load_data_window(ifilepath, system, window=30*6):
    """ this can load latency of the merged files, and each latency point is the average of a window  
    to generate merged file, run: 
    cat client.latency.firstByte* > client.latency.firstByte.all; sort -n -k 1 client.latency.firstByte.all > client.latency.firstByte.all.sort

    """

    server_lat = [[] for _ in range(10)]
    server_window_lat_list = defaultdict(list)
    lat_large_obj, lat_small_obj, lat_all = [], [], []

    start_ts, final_ts = 0, 0
    last_ts = 0
    window_lat_list_large_obj, window_lat_list_small_obj = [], []
    window_lat_list = []

    with open(ifilepath) as ifile:
        for line in ifile:
            ts, req_str, latency = line.strip().split()
            ts, latency = int(ts), float(latency)
            obj, size = [int(f) for f in req_str.split("_")]
            bucket = obj % 200 
            large_obj = size > 128*1024 

            if start_ts == 0:
                start_ts = ts
                last_ts = ts
            final_ts = ts

            window_lat_list.append(latency)
            server_window_lat_list[bucket_to_server_map[bucket][0]].append(latency)
            # server_window_lat_list[bucket_to_server_map[bucket][1]].append(latency)
            if large_obj:
                window_lat_list_large_obj.append(latency)
            else:
                window_lat_list_small_obj.append(latency)

            if ts - last_ts > window:
                lat_all.append(sum(window_lat_list)/len(window_lat_list))
                window_lat_list.clear()

                if len(window_lat_list_large_obj) != 0:
                    lat_large_obj.append(sum(window_lat_list_large_obj)/len(window_lat_list_large_obj))
                # else:
                    # lat_large_obj.append(lat_large_obj[-1])
                window_lat_list_large_obj.clear()

                if len(window_lat_list_small_obj) != 0:
                    lat_small_obj.append(sum(window_lat_list_small_obj)/len(window_lat_list_small_obj))
                # else:
                #     lat_small_obj.append(lat_small_obj[-1])
                window_lat_list_small_obj.clear()

                for i in range(10):
                    server_lat[i].append(np.mean(server_window_lat_list[i]))

                last_ts = ts 



    return final_ts-start_ts, server_lat, lat_large_obj, lat_small_obj, lat_all


def plot_lat_over_time(ifile_dir_list, system_list, obj_type="all", scaling=10):
    for lat_type in ("firstByte", "fullResp"): 
        for obj_type in ("all", "large", "small"): 
            for i in range(len(ifile_dir_list)):
                ifile_dir = ifile_dir_list[i]
                system = system_list[i]
                data_path = f"{ifile_dir}/client/c2dn/output/client.latency.{lat_type}.all.sort"
                time_span, server_lat, lat_large_obj, lat_small_obj, lat_all = load_data_window(data_path, system) 
                lat_list = {"all": lat_all, "large": lat_large_obj, "small": lat_small_obj}[obj_type]
                plt.plot(np.linspace(0, time_span*scaling, len(lat_list)-2)/3600, lat_list[2:], label=system)

            plt.xlabel("Time(hour)")
            plt.ylabel("Latency (ms)")
            plt.grid(linestyle="--")
            plt.legend()
            plt.savefig(f"fig/{lat_type}_{obj_type}.pdf", bbox_inches="tight")
            plt.clf()


def plot_server_lat_over_time(ifile_dir_list, system_list, name, scaling=10):
    for lat_type in ("firstByte", ): 
        for server in range(10):
            for i in range(len(ifile_dir_list)):
                ifile_dir = ifile_dir_list[i]
                system = system_list[i]
                data_path = f"{ifile_dir}/client/c2dn/output/client.latency.{lat_type}.all.sort"
                time_span, server_lat, lat_large_obj, lat_small_obj, lat_all = load_data_window(data_path, system, 30*2) 
                lat_list = server_lat[server]
                plt.plot(np.linspace(0, time_span*scaling, len(lat_list)-2)/3600, lat_list[2:], label=system)

            plt.xlabel("Time(hour)")
            plt.ylabel("Latency (ms)")
            plt.grid(linestyle="--")
            plt.legend()
            plt.savefig(f"fig/{lat_type}_{name}_{server}.pdf", bbox_inches="tight")
            plt.clf()

def plot_lat_cdf(ifile_dir_list, system_list, ): 
    for lat_type in ("firstByte", "fullResp", ): 
        for obj_type in ("all", "large", "small"): 
            for i in range(len(ifile_dir_list)):
                ifile_dir = ifile_dir_list[i]
                system = system_list[i]
                data_path = f"{ifile_dir}/client/c2dn/output/client.latency.{lat_type}.all.sort"
                time_span, lat_large_obj, lat_small_obj, lat_all = load_data(data_path, system) 
                print("load {} time span {}".format(data_path, time_span))
                lat_list = {"all": lat_all, "large": lat_large_obj, "small": lat_small_obj}[obj_type]
                x, y = conv_to_cdf(lat_list)
                plt.plot(x, y, label=system)

            plt.xlabel("Latency (ms)")
            plt.ylabel("Fraction (CDF)")
            plt.xscale("log")
            if lat_type == "firstByte":
                plt.xlim(12, 600)
            plt.xticks([20, 50, 100, 500], [20, 50, 100, 500])
            # plt.xlim(0, 500)
            plt.grid(linestyle="--")
            plt.legend()
            plt.savefig(f"fig/{lat_type}_cdf_{obj_type}.pdf", bbox_inches="tight")
            plt.clf()


if __name__ == "__main__":
    BASE_DIR = "/disk/"
    dir1 = f"{BASE_DIR}/0130/aws_CDN_akamai1_expLatency_unavail0_100G/"
    dir2 = f"{BASE_DIR}/0130/aws_CDN_akamai1_expLatency_unavail1_100G/"

    # plot_lat_cdf(
    #     [
    #     f"/{BASE_DIR}/0124/aws_CDN_akamai2_expLatency_unavail0_1000G/", 
    #     f"/{BASE_DIR}/0124/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/",
    #      ], 
    #      ["CDN", "Donut"], 
    #     )

    # plot_lat_cdf(
    #     [
    #     f"/{BASE_DIR}/0125/aws_CDN_akamai2_expLatency_unavail1_1000G/", 
    #     f"/{BASE_DIR}/0125/aws_C2DN_akamai2_expLatency_unavail1_43_1000G/",
    #      ], 
    #      ["CDN", "Donut"], 
    #     )


    # plot_lat_cdf(
    #     [
    #     f"/{BASE_DIR}/0201/aws_CDN_akamai2_expLatency_unavail0_1000G/", 
    #     f"/{BASE_DIR}/0201/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/",
    #      ], 
    #      ["CDN", "Donut"], 
    #     )





    # plot_lat_cdf(
    #     [
    #     f"/{BASE_DIR}/0127/aws_CDN_akamai1_expLatency_unavail0_100G/", 
    #     f"/{BASE_DIR}/0127/aws_C2DN_akamai1_expLatency_unavail0_43_100G/",
    #      ], 
    #      ["CDN", "C2DN"], 
    #     )

    # plot_lat_cdf(
    #     [
    #     f"/{BASE_DIR}/0130/aws_CDN_akamai1_expLatency_unavail0_100G/", 
    #     f"/{BASE_DIR}/0130/aws_C2DN_akamai1_expLatency_unavail0_43_100G/",
    #      ], 
    #      ["CDN", "C2DN"], 
    #     )





    # plot_lat_over_time(
    #     [
    #     f"/{BASE_DIR}/0130/aws_CDN_akamai1_expLatency_unavail0_100G/", 
    #     f"/{BASE_DIR}/0130/aws_CDN_akamai1_expLatency_unavail1_100G/",
    #      ], 
    #      ["CDN", "CDN_un"], scaling=10, 
    #     )

    # plot_lat_over_time(
    #     [
    #     f"/{BASE_DIR}/0124/aws_CDN_akamai2_expLatency_unavail0_1000G/", 
    #     f"/{BASE_DIR}/0124/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/",
    #      ], 
    #      ["CDN", "C2DN"], scaling=10, 
    #     )

    # plot_lat_over_time(
    #     [
    #     f"/{BASE_DIR}/0124/aws_CDN_akamai2_expLatency_unavail0_1000G/", 
    #     f"/{BASE_DIR}/0125/aws_CDN_akamai2_expLatency_unavail1_1000G/", 
    #      ], 
    #      ["CDN", "CDN_un"], scaling=10, 
    #     )




    plot_server_lat_over_time(
        [
        dir1, 
        dir2, 
         ], 
         ["CDN", "CDN_un"], "CDN", scaling=10, 
        )


    # plot_server_lat_over_time(
    #     [
    #     f"/{BASE_DIR}/0124/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/",
    #     f"/{BASE_DIR}/0125/aws_C2DN_akamai2_expLatency_unavail1_43_1000G/",
    #      ], 
    #      ["C2DN", "C2DN_un"], "C2DN", scaling=10, 
    #     )










