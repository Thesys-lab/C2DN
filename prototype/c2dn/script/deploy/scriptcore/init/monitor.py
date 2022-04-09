import os
import psutil
import time
import subprocess


MB=1000*1000
GB=1000*1000
OUTPUT_DIR="/tmp/c2dn/"


def init(): 
    # subprocess.run("pkill python2", shell=True)
    psutil.net_io_counters.cache_clear()
    if not os.path.exists(OUTPUT_DIR + "/stat"):
        os.makedirs(OUTPUT_DIR + "/stat")
    if os.path.exists(OUTPUT_DIR + "/stat/dstat.csv"):
        os.remove(OUTPUT_DIR + "/stat/dstat.csv")


def get_proc():
    p_ts = [p for p in psutil.process_iter(attrs=['pid', 'name']) if 'TS' in p.info['name']]
    p_fe = [p for p in psutil.process_iter(attrs=['pid', 'name']) if 'frontend' in p.info['name']]
    p_client = [p for p in psutil.process_iter(attrs=['pid', 'name']) if 'client' in p.info['name']]
    p_origin = [p for p in psutil.process_iter(attrs=['pid', 'name']) if 'origin' in p.info['name']]

    proc_list = []
    for p_list in [p_ts, p_fe, p_client, p_origin]:
        if len(p_list) == 1:
            proc_list.append(p_list[0])
        else:
            print(f"proc list {p_list} {proc_list}")
    return proc_list

def dump_host_stat(cmd, msg):
    try:
        with open(OUTPUT_DIR + "/stat/host", "a") as ofile:
            ofile.write("{} >>>>>>>>>>>>>>>>>>>>>>>> {} {}\n".format(time.strftime("%H:%M:%S", time.localtime(time.time())), msg, cmd))
            p = subprocess.run(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            ofile.write(p.stdout.decode() + p.stderr.decode())
    except Exception as e:
        print(e)




    



def monitor():
    proc_list = get_proc()
    for p in proc_list:
        p.cpu_percent()
    time.sleep(5)

    state = "run"
    ofile = open(OUTPUT_DIR + "/stat/proc", "w")
    ofile_sys = open(OUTPUT_DIR + "/stat/sys", "w")

    while state == "run":
        ofile.write("{}:".format(time.time()))
        ofile_sys.write("{}:".format(time.time()))
        for p in proc_list:
            mem = p.memory_full_info()
            cpu = p.cpu_percent()
            io_counters = p.io_counters()
            cpu_times = p.cpu_times()
            ofile.write("{},{},{},{},{}| ".format(p.info["name"], cpu, list(["{:.2f}".format(i/MB) for i in mem]), io_counters[:], cpu_times[:]))
        ofile.write("\n")
        ofile.flush()

        try:
            ofile_sys.write("{},{}\n".format(psutil.net_io_counters(pernic=True, nowrap=True), psutil.disk_io_counters()))
            ofile_sys.flush()
        except Exception as e:
            print(e)

        time.sleep(10)
        proc_list = get_proc()
        with open(OUTPUT_DIR + "/monitorCmd") as ifile:
            state=ifile.read().strip()

    ofile.close()
    ofile_sys.close()


def run():
    init() 
    
    dump_host_stat("ifconfig", "start")
    dump_host_stat("nstat", "start")
    dump_host_stat("vmstat -w -d -n", "start")
    # p = subprocess.Popen(["/usr/bin/dstat", "-d", "--disk-util", "--disk-tps", "--disk-avgqu", "--disk-wait", "--net", "--output", OUTPUT_DIR + "/stat/dstat.csv", "1"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    time.sleep(20)
    try:
        monitor()
    except Exception as e:
        print(e)
    # try:
    #     p.terminate()
    # except Exception as e:
    #     print(e)

    dump_host_stat("ifconfig", "end")
    dump_host_stat("vmstat -w -d -n", "end")
    dump_host_stat("nstat", "end")


if __name__ == "__main__":
    run()

