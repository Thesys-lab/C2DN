//
//  simulator.cpp
//  CDNSimulator
//
//  Created by Juncheng on 11/18/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#include <assert.h>
#include "simulator.hpp"
#include "libCacheSim.h"

using namespace CDNSimulator;
using namespace std;

simulator_arg_t CDNSimulator::default_sim_arg() {
    simulator_arg_t sim_arg;

    const time_t t = time(0);
    tm *ltm = localtime(&t);

    sim_arg.EC_n = 0;
    sim_arg.EC_k = 0;
    sim_arg.EC_size_threshold = 0;
    sim_arg.admission = 0;
    sim_arg.log_interval = 300;
    sim_arg.gutter_space = 0;
    sim_arg.cluster_mode = invalid_mode;
    sim_arg.check_one_more = false;
    sim_arg.parity_rebalance = false;
    sim_arg.log_folder = "log_" + std::to_string(ltm->tm_mon + 1) + "_" +
                         std::to_string(ltm->tm_mday);
    mkdir(sim_arg.log_folder.c_str(), 0770);

    sim_arg.trace_type = BIN_TRACE;
    sim_arg.reader_init_params = {.real_time_field=1, .obj_id_field=2, .obj_size_field=3};

    return sim_arg;
}

void CDNSimulator::parse_cmd_arg(int argc, char **argv,
                                 simulator_arg_t *sim_arg) {

    std::string trace_type;
    std::string server_cache_size;

    namespace po = boost::program_options;
    po::options_description desc("Options");
    desc.add_options()("help,h", "Print help messages")(
            "name", po::value<std::string>(&(sim_arg->exp_name))->default_value("cluster"), "exp name")(
            "alg,a", po::value<std::string>(&(sim_arg->cache_alg)), "cache replacement algorithm")(
            "dataPath,d", po::value<std::string>(&(sim_arg->trace_path))->required(), "data path")(
            "serverCacheSize,s", po::value<std::string>(&server_cache_size)->required(),
            "per server cache size (avg if different)")(
            "serverCacheSizes", po::value<std::vector<std::string>>()->multitoken(), "list of cache sizes")(
            "nServer,m", po::value<unsigned long>(&(sim_arg->n_server))->required(), "the number of cache servers")(
            "traceFormat,t", po::value<std::string>(&(sim_arg->trace_format_str))->required(),
            "the format of trace[akamai1b/akamai1bWithBucket]")(
            "logInterval,l", po::value<unsigned long>(&(sim_arg->log_interval)),
            "the log output interval in virtual time")(
            "EC_n,n", po::value<unsigned int>(&(sim_arg->EC_n))->required(), "number data and parity chunks in EC")(
            "EC_k,k", po::value<unsigned int>(&(sim_arg->EC_k))->required(), "number of data chunks in EC")(
            "admission,o", po::value<unsigned int>(&(sim_arg->admission))->default_value(0), "n-hit-wonder filters")(
            "EC_sizeThreshold,z", po::value<unsigned int>(&(sim_arg->EC_size_threshold))->default_value(0),
            "size threshold for coding")(
            "checkOneMore", po::value<bool>(&(sim_arg->check_one_more)), "whether check one more server")(
            "rebalance,b", po::value<bool>(&(sim_arg->parity_rebalance)), "whether use parity to rebalance write")(
            "clusterMode", po::value<std::string>(&(sim_arg->cluster_mode_str)),
            "mode of cluster, support two_rep_popularity, two_rep_always, no_rep, C2DN")(
            "failureData,f", po::value<std::string>(&(sim_arg->failure_data))->default_value(""),
            "Path to failure data");
    po::variables_map vm;
    try {
        po::store(po::parse_command_line(argc, argv, desc), vm);

        if (vm.count("help")) {
            std::cout << desc << std::endl;
            exit(0);
        }
        po::notify(vm); // throws on error, so do after help in case

        sim_arg->server_cache_size = Utils::convert_size(server_cache_size);
        sim_arg->server_cache_sizes = new unsigned long[sim_arg->n_server];

        sim_arg->server_weight = new uint32_t[sim_arg->n_server];
        std::fill_n(sim_arg->server_weight, sim_arg->n_server, 1);

        if (!vm["serverCacheSizes"].empty()) {
            std::vector<std::string> cache_sizes;
            cache_sizes = vm["serverCacheSizes"].as<std::vector<std::string>>();

            if (cache_sizes.size() == 1) {
                std::vector<std::string> cache_sizes_new;
                boost::split(cache_sizes_new,
                             cache_sizes.at(0),
                             [](char c) { return c == ' '; });
                cache_sizes = cache_sizes_new;
            }
            DEBUG("%lu server cache sizes, first one %s\n",
                  cache_sizes.size(),
                  cache_sizes[0].c_str());

            long sum_cache_size = 0;
            for (unsigned long i = 0; i < sim_arg->n_server; i++) {
                sum_cache_size += Utils::convert_size(cache_sizes.at(i));
                sim_arg->server_cache_sizes[i] = Utils::convert_size(cache_sizes.at(i));
            }
            if (std::abs(sum_cache_size - (long) (sim_arg->server_cache_size * sim_arg->n_server)) > 1000) {
                std::cerr << "sum of given cache sizes "
                          << sum_cache_size / GB << "GB (" << sum_cache_size << ")"
                          << " is not the same as avg cache size " << sim_arg->server_cache_size / GB
                          << "GB - " << server_cache_size << "* n_server (" << sim_arg->n_server << ")" << std::endl;
                exit(0);
            }
            for (unsigned long i = 0; i < sim_arg->n_server; i++) {
                sim_arg->server_weight[i] = sim_arg->server_cache_sizes[i];
            }

        } else {
            INFO("fill all servers with same cache size %ld GB\n", sim_arg->server_cache_size / GB);
            std::fill_n(sim_arg->server_cache_sizes, sim_arg->n_server, sim_arg->server_cache_size);
            std::fill_n(sim_arg->server_weight, sim_arg->n_server, 1);
        }
    } catch (po::error &e) {
        std::cerr << "ERROR: " << e.what() << std::endl << std::endl;
        std::cerr << desc << std::endl;
        exit(1);
    }

    INFO("trace: %s, cache alg %s, cache_size %lu, n_server %lu, trace_type %s, "
         "n %d, k %d, "
         "admission %u, EC_size_threshold %u, "
         "gutter_space: %.4lf, "
         "check one more %d, "
         "rebalance %d, cluster mode %s,"
         "failure_data %s\n",
         sim_arg->trace_path.c_str(),
         sim_arg->cache_alg.c_str(),
         sim_arg->server_cache_size,
         sim_arg->n_server,
         sim_arg->trace_format_str.c_str(),
         sim_arg->EC_n,
         sim_arg->EC_k,
         sim_arg->admission,
         sim_arg->EC_size_threshold,
         sim_arg->gutter_space,
         sim_arg->check_one_more,
         sim_arg->parity_rebalance,
         sim_arg->cluster_mode_str.c_str(),
         sim_arg->failure_data.c_str());
}

void CDNSimulator::prepare_arg(simulator_arg_t *sim_arg) {
    if (sim_arg->trace_format_str == "akamai1b") {
        sim_arg->trace_format = akamai1b;
        memcpy(sim_arg->reader_init_params.binary_fmt, "III", 4);
    } else {
        ERROR("unknown_obj_type trace type %s\n", sim_arg->trace_format_str.c_str());
        abort();
    }

    std::vector<std::string> split_results;
    boost::split(split_results,
                 sim_arg->trace_path,
                 [](char c) { return c == '/'; });

    sim_arg->exp_name = sim_arg->exp_name + "_" + split_results.back() + "_" +
                        sim_arg->cache_alg + "_" +
                        std::to_string(sim_arg->server_cache_size * sim_arg->n_server / GB) + "GB" +
                        "_m" + std::to_string(sim_arg->n_server) +
                        "_o" + std::to_string(sim_arg->admission) +
                        "_n" + std::to_string(sim_arg->EC_n) +
                        "_k" + std::to_string(sim_arg->EC_k) +
                        "_z" + std::to_string(sim_arg->EC_size_threshold) +
                        "_" + sim_arg->cluster_mode_str;
    if (sim_arg->gutter_space > 0.000001)
        sim_arg->exp_name += "_g" + std::to_string(sim_arg->gutter_space);

    if (sim_arg->check_one_more)
        sim_arg->exp_name += "_checkOneMore";

    if (sim_arg->parity_rebalance)
        sim_arg->exp_name += "_rebalance";

    if (!sim_arg->failure_data.empty())
        sim_arg->exp_name += "_f_" + sim_arg->failure_data;

    if (sim_arg->cluster_mode_str == "no_rep") {
        sim_arg->cluster_mode = no_replication;
        assert(sim_arg->EC_n == 1);
        assert(sim_arg->EC_k == 1);
    } else if (sim_arg->cluster_mode_str == "two_rep_popularity") {
        sim_arg->cluster_mode = two_rep_popularity;
        assert(sim_arg->EC_n == 2);
        assert(sim_arg->EC_k == 1);
    } else if (sim_arg->cluster_mode_str == "two_rep_always") {
        sim_arg->cluster_mode = two_rep_always;
        assert(sim_arg->EC_n == 2);
        assert(sim_arg->EC_k == 1);
    } else if (sim_arg->cluster_mode_str == "three_rep_always") {
        sim_arg->cluster_mode = three_rep_always;
        assert(sim_arg->EC_n == 3);
        assert(sim_arg->EC_k == 1);
    } else if (sim_arg->cluster_mode_str == "C2DN") {
        sim_arg->cluster_mode = C2DN;
        assert(sim_arg->EC_n > 2);
        assert(sim_arg->EC_k >= 2);
    } else if (sim_arg->cluster_mode_str == "C2DN_add_n") {
        sim_arg->cluster_mode = C2DN_add_n;
        assert(sim_arg->EC_n > 2);
        assert(sim_arg->EC_k >= 2);
    } else if (sim_arg->cluster_mode_str == "C2DN_add_n_three_rep") {
      sim_arg->cluster_mode = C2DN_add_n;
      assert(sim_arg->EC_n > 2);
      assert(sim_arg->EC_k >= 2);
    }
    assert(sim_arg->cluster_mode != invalid_mode);
}

std::vector<std::string> get_traces_from_config(const std::string &config_loc) {

    std::ifstream ifs(config_loc, std::ios::in);
    std::vector<std::string> traces;
    std::string line;
    while (getline(ifs, line)) {
        traces.push_back(line);
    }

    ifs.close();
    return traces;
}

void CDNSimulator::run(simulator_arg_t *sim_arg) {
    reader_t *reader = setup_reader(
            sim_arg->trace_path.c_str(),
            sim_arg->trace_type,
            sim_arg->obj_id_type,
            &sim_arg->reader_init_params);

    /* threads for cacheServerThread and cacheClusterThread */
    std::thread *t;
    std::vector<std::thread *> threads;

    cacheServer *cache_servers[sim_arg->n_server];
    cacheCluster *cache_cluster;
    cacheClusterThread *cache_cluster_thread;

    server_params_t server_params;
    server_params.cache_alg = sim_arg->cache_alg;
    server_params.cache_alg_params = nullptr;
    server_params.gutter_space = sim_arg->gutter_space;
    server_params.EC_n = sim_arg->EC_n;
    server_params.EC_k = sim_arg->EC_k;

    cluster_params_t cluster_params = {
            .cluster_id = 1,
            .exp_name = sim_arg->exp_name,
            .trace_format = sim_arg->trace_format,
            .cache_servers = cache_servers,
            .n_server = sim_arg->n_server,
            .server_cache_sizes = sim_arg->server_cache_sizes,
            .server_weight = sim_arg->server_weight,
            .admission = sim_arg->admission,
            // .ICP = sim_arg->ICP,
            .check_one_more = sim_arg->check_one_more,
            .parity_rebalance = sim_arg->parity_rebalance,

            .cluster_mode = sim_arg->cluster_mode,
            .gutter_space = sim_arg->gutter_space,
    };

    EC_params_t ec_params = {
            .n = sim_arg->EC_n,
            .k = sim_arg->EC_k,
            .size_threshold = sim_arg->EC_size_threshold,

    };

    try {
        /* initialize cacheServers */
        for (unsigned long i = 0; i < sim_arg->n_server; i++) {
            server_params.server_id = i;
            server_params.cache_size = sim_arg->server_cache_sizes[i];
            server_params.server_name = "server_" + std::to_string(i);

            cache_servers[i] = new cacheServer(server_params);
        }

        /* initialize cacheCluster and cacheClusterThread */
        cache_cluster = new cacheCluster(cluster_params, ec_params);
        cache_cluster_thread = new cacheClusterThread(sim_arg->param_string,
                                                      sim_arg->exp_name,
                                                      cache_cluster,
                                                      reader,
                                                      sim_arg->failure_data,
                                                      sim_arg->log_folder,
                                                      sim_arg->log_interval);

        /* build thread for cacheClusterThread */
        t = new std::thread(&cacheClusterThread::run, cache_cluster_thread);
        threads.push_back(t);

        INFO("main thread finishes initialization, begins waiting for all cache "
             "clusters\n");
        /* wait for cache server and cache layer to finish computation */
        for (auto it = threads.begin(); it < threads.end(); it++) {
            (**it).join();
            delete *it;
        }

        /* free cacheServerThread and cacheClusterThread */
        for (auto cache_server : cache_servers) {
            delete cache_server;
        }
        delete cache_cluster;
        delete cache_cluster_thread;

    } catch (std::exception &e) {
        std::cerr << e.what() << std::endl;
        print_stack_trace();
    }
    close_reader(reader);
}

int main(int argc, char *argv[]) {
    DEBUG("debug output enabled\n");
    VERBOSE("verbose output enabled\n");

    try {
        CDNSimulator::simulator_arg_t sim_arg = default_sim_arg();

        /* save the parameter string with results */
        for (long i = 1; i < argc; i++)
            sim_arg.param_string += std::string(argv[i]) + " ";

        parse_cmd_arg(argc, argv, &sim_arg);

        prepare_arg(&sim_arg);

        CDNSimulator::run(&sim_arg);

    } catch (std::exception &e) {
        std::cerr << e.what() << std::endl;
        print_stack_trace();
    }

    return 0;
}
