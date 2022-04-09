

import os, sys
sys.path.append(os.path.expanduser("~/workspace/"))
from pyutils.common import * 


def load_proc_stat_data(ifile_dir, system):
    """ this loads per process overhead data """
    # early version of psutils does not have iowait for cputimes (the last number in the last part)
    regex = re.compile(r"(?P<proc>[a-zA-Z_\[\]]+),(?P<cpu>\d+\.\d+),\['(?P<rss>\d+\.\d+)', '(?P<vms>\d+\.\d+)', '(?P<shared>\d+\.\d+)', '(?P<text>\d+\.\d+)', '(?P<lib>\d+\.\d+)', '(?P<data>\d+\.\d+)', '(?P<dirty>\d+\.\d+)', '(?P<uss>\d+\.\d+)', '(?P<pss>\d+\.\d+)', '(?P<swap>\d+\.\d+)'\],\((?P<AccuReadCnt>\d+), (?P<AccuWriteCnt>\d+), (?P<AccuReadByte>\d+), (?P<AccuWriteByte>\d+), (?P<AccuReadChar>\d+), (?P<AccuWriteChar>\d+)\),\((?P<user>\d+\.\d+), (?P<sys>\d+\.\d+), (?P<childUser>\d+\.\d+), (?P<childSys>\d+\.\d+)(, (?P<iowait>\d+\.\d+))?\)")
    mem = {"ts":[], "fe":[], "client":[], "origin": []}         # in MB
    cpu = {"ts":[], "fe":[], "client":[], "origin": []}
    cpu_times = {"ts":([], []), "fe":([], []), "client":([], []), "origin": ([], [])}
    # io = {"ts":([],[],[],[],[],[]), "fe":([],[],[],[],[],[]), "client":([],[],[],[],[],[]), "origin": ([],[],[],[],[],[])}
    io = defaultdict(list)
    total_io = defaultdict(float)
    last_accu_io = defaultdict(float)

    accu_read_cnt, accu_write_cnt, accu_read_byte, accu_write_byte, accu_read_char, accu_write_char = [0]*10,[0]*10,[0]*10,[0]*10,[0]*10,[0]*10
    overall_mem, overall_cpu = [], []
    overall_cpu_sys, overall_cpu_user = [], []
    n_read_bytes_server = [0] * 10
    n_write_bytes_server = [0] * 10

    for i in range(10):
        with open("{}/cdn{}/c2dn/stat/proc".format(ifile_dir, i)) as ifile:
            firstline, lastline = None, None
            last_cpu_sys_user_ts, last_cpu_sys_user_fe = (0,0), (0,0)
            cpu_sys_user_ts, cpu_sys_user_fe = (0,0), (0,0)
            last_ts = 0
            n_pts = 0

            #### calculate the CPU and memory usage over time ####
            for line in ifile:
                n_pts += 1
                ts, line = line.split(":")
                ts = float(ts)
                proc_data_list = line.split("|")
                mem_ts, mem_fe, cpu_ts, cpu_fe = 0, 0, 0, 0

                if firstline is None:
                    firstline = line
                    first_ts = ts
                lastline = line


                for per_proc_data in proc_data_list:
                    # [TS_MAIN],92.7,['3308.83', '12138.01', '13.13', '4.64', '0.00', '3729.83', '0.00', '3325.98', '3327.96', '0.00'],(916811604, 233306627, 627057999872, 3102793605120, 3701958721023, 6802716622344),(5973.53, 9033.7, 0.0, 0.0)
                    per_proc_data = per_proc_data.strip()
                    if len(per_proc_data) == 0:
                        continue
                    m = regex.match(per_proc_data)
                    if m is None:
                        print("cannot match " + per_proc_data)
                    proc = m.group("proc").replace("[TS_MAIN]", "ts").replace("frontend", "fe")

                    if proc == "ts":
                        mem_ts = float(m.group("rss"))/1000
                        cpu_ts = float(m.group("cpu"))/100
                        if last_cpu_sys_user_ts != (0,0):
                            cpu_sys_user_ts = ((float(m.group("sys"))-last_cpu_sys_user_ts[0])/(ts-last_ts),
                                (float(m.group("user"))-last_cpu_sys_user_ts[1])/(ts-last_ts))
                        last_cpu_sys_user_ts = (float(m.group("sys")), float(m.group("user")))


                    elif proc == "fe":
                        mem_fe = float(m.group("rss"))/1000
                        cpu_fe = float(m.group("cpu"))/100
                        if last_cpu_sys_user_fe != (0,0):
                            cpu_sys_user_fe = ((float(m.group("sys"))-last_cpu_sys_user_fe[0])/(ts-last_ts),
                                (float(m.group("user"))-last_cpu_sys_user_fe[1])/(ts-last_ts))
                        last_cpu_sys_user_fe = (float(m.group("sys")), float(m.group("user")))

                        overall_mem.append(mem_ts + mem_fe)
                        overall_cpu.append(cpu_ts + cpu_fe)
                        overall_cpu_sys.append(cpu_sys_user_ts[0] + cpu_sys_user_fe[0])
                        overall_cpu_user.append(cpu_sys_user_ts[1] + cpu_sys_user_fe[1])

                    # disk 
                    # if proc == "ts" and last_ts == 0:
                    #     print(m.groups())

                    if proc == "ts":
                        if last_ts != 0:
                            # print(last_accu_io)
                            read_cnt = float(m.group("AccuReadCnt")) - last_accu_io["AccuReadCnt"]
                            write_cnt = float(m.group("AccuWriteCnt")) - last_accu_io["AccuWriteCnt"]
                            # this is from real storage layer 
                            read_byte = float(m.group("AccuReadByte")) - last_accu_io["AccuReadByte"]
                            write_byte = float(m.group("AccuWriteByte")) - last_accu_io["AccuWriteByte"]
                            # this includes bytes hit on pagecache
                            read_byte_char = float(m.group("AccuReadChar")) - last_accu_io["AccuReadChar"]
                            write_byte_char = float(m.group("AccuWriteChar")) - last_accu_io["AccuWriteChar"]

                            if read_byte < 0 or write_byte < 0 or read_byte > 40*GB or write_byte > 20*GB or read_byte/(ts - last_ts)/MB > 10000 or write_byte/(ts - last_ts)/MB > 10000:
                                print("server {} ts {:.2f}, read {:.2f}GB, write {:.2f}GB".format(i, ts - last_ts, read_byte/GB, write_byte/GB))

                            if read_cnt > 0 and read_byte > 1*MB: 
                                io["read_kops"].append(read_cnt/(ts - last_ts)/1000)
                                io["write_kops"].append(write_cnt/(ts - last_ts)/1000)
                                io["read_MBs"].append(read_byte/(ts - last_ts)/MB)
                                io["write_MBs"].append(write_byte/(ts - last_ts)/MB)
                                io["read_MBs_char"].append(read_byte_char/(ts - last_ts)/MB)
                                io["write_MBs_char"].append(write_byte_char/(ts - last_ts)/MB)

                                # print("ts {:.0f} server {} read MB {:.2f} write MB {:.2f}".format(ts-first_ts, i, read_byte/(ts - last_ts)/MB, write_byte/(ts - last_ts)/MB))

                                total_io["read_cnt"] += read_cnt
                                total_io["write_cnt"] += write_cnt
                                total_io["read_byte"] += read_byte
                                total_io["write_byte"] += write_byte
                                total_io["read_byte_char"] += read_byte_char
                                total_io["write_byte_char"] += write_byte_char
                                n_read_bytes_server[i] += read_byte
                                n_write_bytes_server[i] += write_byte

                        last_accu_io["AccuReadCnt"] = float(m.group("AccuReadCnt"))
                        last_accu_io["AccuWriteCnt"] = float(m.group("AccuWriteCnt"))
                        last_accu_io["AccuReadByte"] = float(m.group("AccuReadByte"))
                        last_accu_io["AccuWriteByte"] = float(m.group("AccuWriteByte"))
                        last_accu_io["AccuReadChar"] = float(m.group("AccuReadChar"))
                        last_accu_io["AccuWriteChar"] = float(m.group("AccuWriteChar"))



                last_ts = ts


    print("proc-level: {} {:.0f}s {} M reads, {} M writes \
        {:.2f} TB read, {:.2f} TB write, \
        {:.2f} TB read char, {:.2f} TB write char".format(
            system, last_ts - first_ts, 
            int(total_io["read_cnt"]/1000/1000), int(total_io["write_cnt"]/1000/1000),
            total_io["read_byte"]/TB, total_io["write_byte"]/TB,
            total_io["read_byte_char"]/TB, total_io["write_byte_char"]/TB
            )
        )
    n_read_bytes_server = np.array(n_read_bytes_server)
    n_write_bytes_server = np.array(n_write_bytes_server)
    # print(n_read_bytes_server/np.min(n_read_bytes_server))
    # print(n_write_bytes_server/np.min(n_write_bytes_server))
    read_GB_server = (n_read_bytes_server/GB).astype(int)
    write_GB_server = (n_write_bytes_server/GB).astype(int)
    print("mean read load in GB across server {}, P99 {} Mbps ".format(read_GB_server, np.percentile(io["read_MBs"], 99)))
    print("mean write load in GB across server {}, P99 {} Mbps".format(write_GB_server, np.percentile(io["write_MBs"], 99)))




    return overall_mem, overall_cpu, overall_cpu_sys, overall_cpu_user, io


def load_sys_overhead_nic(dat_path, platform="aws"):
    """ calculate and return a list of cluster bandwidth usages """ 

    nic_name = "en"
    if platform == "cloudlab":
        nic_name = "eno1d1"

    regex = re.compile(r"'(?P<nic>\w+)': snetio\(bytes_sent=(?P<byteSend>\d+), bytes_recv=(?P<byteRecv>\d+), packets_sent=(?P<pktSend>\d+), packets_recv=(?P<pktRecv>\d+), errin=\d+, errout=\d+, dropin=\d+, dropout=\d+\)")
    intra_traffic_ts_dict = defaultdict(int)
    intra_traffic_list = []
    for i in range(10):
        fname = "{}/stat/sysStat.{}".format(dat_path, i)
        if not os.path.exists(fname):
            fname = "{}/stat/sysStat2.{}".format(dat_path, i)
        last_send, last_recv, last_ts = 0, 0, 0
        with open(fname) as ifile:
            for line in ifile:
                ts = float(line.split(":")[0])
                for m in  regex.finditer(line):
                    if "lo" in m.group("nic"):
                        continue
                    elif nic_name in m.group("nic"):
                        if last_send == 0 and last_recv == 0:
                            last_send = int(m.group("byteSend"))/GB
                            last_recv = int(m.group("byteRecv"))/GB
                            last_ts = ts
                        else:
                            cur_send, cur_recv = int(m.group("byteSend"))/GB, int(m.group("byteRecv"))/GB
                            bandwidth = (cur_send + cur_recv - last_send - last_recv)/(ts-last_ts)/2 * 8
                            intra_traffic_list.append(bandwidth)
                            # round each ts to the closest 5s
                            intra_traffic_ts_dict[ts//5*5] += bandwidth
                            last_send, last_recv = cur_send, cur_recv
                            last_ts = ts
                    else:
                        if platform != "cloudlab":
                            raise RuntimeError("cannot find a match in {}".format(line))
                        continue
                    # print(i, m.group("nic"), int(m.group("byteSend"))/GB, int(m.group("byteRecv"))/GB)

    # print(list(zip(*sorted(intra_traffic_ts_dict.items(), key=lambda x:x[0])))[1][:20])
    cluster_bandwidth = list(zip(*sorted(intra_traffic_ts_dict.items(), key=lambda x:x[0])))[1]
    per_server_bandwidth = intra_traffic_list

    return per_server_bandwidth


def load_sys_overhead_IO_dat(dat_path, start_ts):
    """ calculate and return a list of storage IO usages 
        return iops and byte/s 
    """ 

    regex_disk = re.compile(r"sdiskio\(read_count=(?P<read_cnt>\d+), write_count=(?P<write_cnt>\d+), read_bytes=(?P<read_byte>\d+), write_bytes=(?P<write_byte>\d+), read_time=(?P<read_time>\d+), write_time=(?P<write_time>\d+), read_merged_count=(?P<read_merge>\d+), write_merged_count=(?P<write_merge>\d+), busy_time=(?P<busy_time>\d+)\)")
    read_iops_list, write_iops_list, read_MB_list, write_MB_list, read_merge_list, write_merge_list = [], [], [], [], [], []
    for i in range(10):
        fname = "{}/cdn{}/c2dn/stat/sys".format(dat_path, i)
        with open(fname) as ifile:
            line = ifile.readline()
            first_ts = float(line.split(":")[0])
            ts = float(line.split(":")[0])
            # print(line)
            m = regex_disk.findall(line)[0]

            last_read_iops, last_write_iops, last_read_byte, last_write_byte, _, _, last_read_merge, last_write_merge, _ = m
            for line in ifile:
                time_interval = float(line.split(":")[0]) - ts
                ts = float(line.split(":")[0])
                if len(line) < 24:
                    continue

                if ts < start_ts:
                    continue 
                # IO
                try:
                    m = regex_disk.findall(line)
                except Exception as e:
                    print("failed to match line {}".format(line))
                if len(m) == 0:
                    raise RuntimeError("failed to match line {}".format(line))
                elif len(m) > 1: 
                    raise RuntimeError("find more than one match {}".format(line))
                m = m[0]

                read_kiops = (float(m[0])-float(last_read_iops))/time_interval/1000
                write_kiops = (float(m[1])-float(last_write_iops))/time_interval/1000
                read_MBs = (float(m[2])-float(last_read_byte))/time_interval/MB
                write_MBs = (float(m[3])-float(last_write_byte))/time_interval/MB
                read_merge_kiops = (float(m[8])-float(last_read_merge))/time_interval/1000
                write_merge_kiops = (float(m[8])-float(last_write_merge))/time_interval/1000
                if read_MBs < 0 or write_MBs < 0 or read_MBs > 10000 or write_MBs > 10000:
                    pass
                    # print("ts {:.0f} server {} read_kiops {:.2f} write_kiops {:.2f} read_MBs {:.2f} write_MBs {:.2f} read_merge_kiops {:.2f} write_merge_kiops {:.2f}".format(
                    #     ts-first_ts, i, read_kiops, write_kiops, read_MBs, write_MBs, read_merge_kiops, write_merge_kiops))
                else: 
                    read_iops_list.append(read_kiops)
                    write_iops_list.append(write_kiops)
                    read_MB_list.append(read_MBs)
                    write_MB_list.append(write_MBs)
                    read_merge_list.append(read_merge_kiops)
                    write_merge_list.append(write_merge_kiops)

                last_read_iops, last_write_iops, last_read_byte, last_write_byte, _, _, last_read_merge, last_write_merge, _ = m

    # print("load_sys_overhead_IO_dat ts {:.0f}s".format(ts - first_ts))
    return read_iops_list, write_iops_list, read_MB_list, write_MB_list, read_merge_list, write_merge_list


def cal_overhead_stat(dat_list):
    dat_stat = [np.mean(dat_list), *list(np.percentile(dat_list, (50, 80, 90, 95, 99, 99.9)))]
    dat_stat = [np.mean(dat_list), *list(np.percentile(dat_list, (50, 80, 90, 95, 99, 99.9))), np.max(dat_list)]
    return dat_stat

def print_stat(name, stat): 
    print("{}: mean {:.2f}, median {:.2f}, P90 {:.2f}, P95 {:.2f}, P99 {:.2f}, P999 {:.2f}".format(name,
        stat[0], stat[1], stat[3], stat[4], stat[5], stat[6], ))

def cmp_stat(stat1, stat2, name): 
    print("{}: mean increases {:.2%}, median {:.2%}, P90 {:.2%}, P95 {:.2%}, P99 {:.2%}, P999 {:.2%}".format(name,
        (stat2[0] - stat1[0])/stat1[0],
        (stat2[1] - stat1[1])/stat1[1],
        (stat2[3] - stat1[3])/stat1[3],
        (stat2[4] - stat1[4])/stat1[4],
        (stat2[5] - stat1[5])/stat1[5],
        (stat2[6] - stat1[6])/stat1[6],
        ))



def plot_CPU_and_io(cdn_dir, c2dn_dir, name):
    """ plot CPU usage box plot """
    _, cdn_cpu, cdn_cpu_sys, cdn_cpu_user, cdn_io = load_proc_stat_data(cdn_dir, "CDN")
    _, c2dn_cpu, c2dn_cpu_sys, c2dn_cpu_user, c2dn_io = load_proc_stat_data(c2dn_dir, "Donut")

    cdn_cpu_stat = cal_overhead_stat(cdn_cpu)
    c2dn_cpu_stat = cal_overhead_stat(c2dn_cpu)
    cmp_stat(cdn_cpu_stat, c2dn_cpu_stat, "CPU")

    # cdn_read_iops_stat, c2dn_read_iops_stat   = cal_overhead_stat(cdn_io["read_kops"]),  cal_overhead_stat(c2dn_io["read_kops"])
    # cmp_stat(cdn_read_iops_stat, c2dn_read_iops_stat, "read_iops")
    # cdn_write_iops_stat, c2dn_write_iops_stat = cal_overhead_stat(cdn_io["write_kops"]), cal_overhead_stat(c2dn_io["write_kops"])
    # cmp_stat(cdn_write_iops_stat, c2dn_write_iops_stat, "write_iops")
    # cdn_read_MBs_stat, c2dn_read_MBs_stat   = cal_overhead_stat(cdn_io["read_MBs"]),  cal_overhead_stat(c2dn_io["read_MBs"])
    # cmp_stat(cdn_read_MBs_stat, c2dn_read_MBs_stat, "read_MBs")
    # cdn_write_MBs_stat, c2dn_write_MBs_stat = cal_overhead_stat(cdn_io["write_MBs"]), cal_overhead_stat(c2dn_io["write_MBs"])
    # cmp_stat(cdn_write_MBs_stat, c2dn_write_MBs_stat, "write_MBs")
    # cdn_read_MBs_char_stat, c2dn_read_MBs_char_stat   = cal_overhead_stat(cdn_io["read_MBs_char"]),  cal_overhead_stat(c2dn_io["read_MBs_char"])
    # cmp_stat(cdn_read_MBs_char_stat, c2dn_read_MBs_char_stat, "read_MBs_char")
    # cdn_write_MBs_char_stat, c2dn_write_MBs_char_stat = cal_overhead_stat(cdn_io["write_MBs_char"]), cal_overhead_stat(c2dn_io["write_MBs_char"])
    # cmp_stat(cdn_write_MBs_char_stat, c2dn_write_MBs_char_stat, "write_MBs_char")

    cdn_read_iops, cdn_write_iops, cdn_read_MB, cdn_write_MB, cdn_read_merge, cdn_write_merge = load_sys_overhead_IO_dat(cdn_dir, 1611482938.4349413)
    c2dn_read_iops, c2dn_write_iops, c2dn_read_MB, c2dn_write_MB, c2dn_read_merge, c2dn_write_merge = load_sys_overhead_IO_dat(c2dn_dir, 1611482345.3552828)
    cdn_read_iops_stat, c2dn_read_iops_stat   = cal_overhead_stat(cdn_read_iops),  cal_overhead_stat(c2dn_read_iops)
    cmp_stat(cdn_read_iops_stat, c2dn_read_iops_stat, "read_iops")
    cdn_write_iops_stat, c2dn_write_iops_stat   = cal_overhead_stat(cdn_write_iops),  cal_overhead_stat(c2dn_write_iops)
    cmp_stat(cdn_write_iops_stat, c2dn_write_iops_stat, "write_iops")
    cdn_read_MB_stat, c2dn_read_MB_stat   = cal_overhead_stat(cdn_read_MB),  cal_overhead_stat(c2dn_read_MB)
    cmp_stat(cdn_read_MB_stat, c2dn_read_MB_stat, "read_byte")
    cdn_write_MB_stat, c2dn_write_MB_stat   = cal_overhead_stat(cdn_write_MB),  cal_overhead_stat(c2dn_write_MB)
    cmp_stat(cdn_write_MB_stat, c2dn_write_MB_stat, "write_byte")

    # print("max {}".format(np.max(cdn_write_MB)))
    # print("max {}".format(np.max(c2dn_write_MB)))
    # print_stat("CDN", cdn_write_MB_stat)
    # print_stat("C2DN", c2dn_write_MB_stat)

    cdn_wmerged_stat, c2dn_wmerged_stat   = cal_overhead_stat(cdn_write_merge),  cal_overhead_stat(c2dn_write_merge)
    cdn_rmerged_stat, c2dn_rmerged_stat   = cal_overhead_stat(cdn_read_merge),  cal_overhead_stat(c2dn_read_merge)
    # cmp_stat(cdn_wmerged_stat, c2dn_wmerged_stat, "write_merge")
    # print(["{:.0f}".format(i) for i in cdn_read_iops_stat], ["{:.0f}".format(i) for i in c2dn_read_iops_stat])
    # print(["{:.0f}".format(i) for i in cdn_write_iops_stat], ["{:.0f}".format(i) for i in c2dn_write_iops_stat])
    # print(["{:.0f}".format(i) for i in cdn_rmerged_stat], ["{:.0f}".format(i) for i in c2dn_rmerged_stat])
    # print(["{:.0f}".format(i) for i in cdn_wmerged_stat], ["{:.0f}".format(i) for i in c2dn_wmerged_stat])


    plt.boxplot((cdn_cpu_sys, c2dn_cpu_sys, cdn_cpu_user, c2dn_cpu_user),
        positions=(1, 1.6, 2.4, 3), sym="", whis=(1, 99.9),
        labels=("CDN", "Donut", "CDN", "Donut"), )
        
    plt.ylabel("CPU usage (# cores)")
    plt.grid(linestyle="--") 
    plt.ylim(bottom=0)
    # plt.ylim(bottom=0, top=2.8)
    plt.text(1, 2.8, "kernel", fontsize=38)
    plt.text(2.5, 2.8, "user", fontsize=38)
    plt.vlines(2, ymin=0, ymax=plt.ylim()[1], linestyle="--", linewidth=2, color='grey')
    plt.savefig(fname=f"fig/{name}cpuSysUser.pdf", bbox_inches="tight")
    plt.clf()



    # plt.boxplot((cdn_io["read_kops"], c2dn_io["read_kops"], cdn_io["write_kops"], c2dn_io["write_kops"]),
    plt.boxplot((cdn_read_iops, c2dn_read_iops, cdn_write_iops, c2dn_write_iops),
        positions=(1, 1.6, 2.4, 3), sym="", whis=(1, 99.9),
        labels=("CDN", "Donut", "CDN", "Donut"), ) # showmeans=True, meanline=True, rotate_xticks=45, 
    plt.ylabel("IOPS (K)")
    plt.grid(linestyle="--") 
    plt.ylim(bottom=0)
    # plt.ylim(bottom=0, top=5.6)
    plt.text(1.1, 5.6, "read", fontsize=38)
    plt.text(2.5, 5.6, "write", fontsize=38)
    plt.vlines(2, ymin=0, ymax=plt.ylim()[1], linestyle="--", linewidth=2, color='grey')
    plt.savefig(fname=f"fig/{name}_diskIOPS.pdf", bbox_inches="tight")
    plt.clf()


    plt.boxplot((cdn_read_MB, c2dn_read_MB, cdn_write_MB, c2dn_write_MB),
        positions=(1, 1.6, 2.4, 3), sym="", whis=(1, 99.9),
        labels=("CDN", "Donut", "CDN", "Donut"), ) # showmeans=True, meanline=True, rotate_xticks=45, 
    plt.ylabel("Bandiwdth (MB/s)")
    plt.grid(linestyle="--") 
    plt.ylim(bottom=0)
    # plt.ylim(bottom=0, top=5.6)
    plt.text(1.1, 720, "read", fontsize=38)
    plt.text(2.5, 720, "write", fontsize=38)
    plt.vlines(2, ymin=0, ymax=plt.ylim()[1], linestyle="--", linewidth=2, color='grey')
    plt.savefig(fname=f"fig/{name}_diskBand.pdf", bbox_inches="tight")
    plt.clf()


# def plot_network(video, web):
#     """ plot CPU usage box plot """
#     video_Gbps_list = load_sys_overhead_nic(video)
#     web_Gbps_list = load_sys_overhead_nic(web)

#     video_stat, web_stat = cal_overhead_stat(video_Gbps_list), cal_overhead_stat(web_Gbps_list)
#     print("{} intra-cluster network: mean {:.2f} Gbps, median {:.2f} Gbps, P90 {:.2f} Gbps, P95 {:.2f} Gbps, P99 {:.2f} Gbps".format("video",
#         video_stat[0], video_stat[1], video_stat[3], video_stat[4], video_stat[5], ))
#     print("{} intra-cluster network: mean {:.2f} Gbps, median {:.2f} Gbps, P90 {:.2f} Gbps, P95 {:.2f} Gbps, P99 {:.2f} Gbps".format("web",
#         web_stat[0], web_stat[1], web_stat[3], web_stat[4], web_stat[5], ))

#     plt.boxplot((video_Gbps_list, web_Gbps_list, ),
#         positions=(1, 2), sym="", whis=(10, 90),
#         labels=("video", "web", ),
#         # ylim=(0,10), yticks=(0,2,4,6,8,10),
#         grid=True, ylabel="Intra-cluster bandwidth (Gbps)")
#     # if name == "web":
#     #     # plt.text(1.05, 0.4, "kernel", fontsize=20)
#     #     # plt.text(2.55, 1.2, "user", fontsize=20)
#     #     plt.text(1, 2.48, "kernel")
#     #     plt.text(2.5, 2.48, "user")
#     #     plt.ylim(top=2.86)
#     # elif name == "video":
#     #     plt.text(1, 2, "kernel")
#     #     plt.text(2.5, 2, "user")
#     # plt.vlines(2, ymin=0, ymax=plt.ylim()[1], linestyle="--", linewidth=2, color='grey')
#     plt.savefig(fname=f"fig/network")
#     plt.clf()


def run(cdn_dir_video, c2dn_dir_video, cdn_dir_web, c2dn_dir_web):

    print("*"*20 + " video ")
    cal_traffic(cdn_dir_video)
    cal_traffic(c2dn_dir_video)
    cal_sys_overhead_dat(cdn_dir_video, platform="aws")
    cal_sys_overhead_dat(c2dn_dir_video, platform="aws")


if __name__ == "__main__":
    BASE_DIR = "/nvme/log/p/2021-01-30/"

    # do not use - this one has a problem with write, use 0125 akamai2 unavail1 for write 
    cdn_dir = f"{BASE_DIR}/0124/aws_CDN_akamai2_expLatency_unavail0_1000G/"
    c2dn_dir = f"{BASE_DIR}/0124/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/"

    cdn_dir = f"{BASE_DIR}/0125/aws_CDN_akamai2_expLatency_unavail1_1000G/"
    c2dn_dir = f"{BASE_DIR}/0125/aws_C2DN_akamai2_expLatency_unavail1_43_1000G/"

    # cdn_dir = f"{BASE_DIR}/0127/aws_CDN_akamai2_expLatency_unavail0_1000G/"
    # c2dn_dir = f"{BASE_DIR}/0127/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/"

    # cdn_dir = f"{BASE_DIR}/0129/aws_CDN_akamai1_expLatency_unavail0_100G/"
    # c2dn_dir = f"{BASE_DIR}/0129/aws_C2DN_akamai1_expLatency_unavail0_43_100G/"

    # cdn_dir = f"{BASE_DIR}/0130/aws_CDN_akamai1_expLatency_unavail0_100G/"
    # c2dn_dir = f"{BASE_DIR}/0130/aws_C2DN_akamai1_expLatency_unavail0_43_100G/"

    cdn_dir = f"{BASE_DIR}/aws_CDN_akamai2_expLatency_unavail0_1000G/"
    c2dn_dir = f"{BASE_DIR}/aws_C2DN_akamai2_expLatency_unavail0_43_1000G/"

    cdn_dir = f"{BASE_DIR}/aws_CDN_akamai1_expLatency_unavail0_100G/"
    c2dn_dir = f"{BASE_DIR}/aws_C2DN_akamai1_expLatency_unavail0_43_100G/"

    plot_CPU_and_io(cdn_dir, c2dn_dir, "video")

    # load_sys_overhead_IO_dat(cdn_dir)
    # load_sys_overhead_IO_dat(c2dn_dir)

