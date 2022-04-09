//
//  simulator.cpp
//  CDNSimulator
//
//  Created by Juncheng on 11/18/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#ifndef SIMULATOR_HPP
#define SIMULATOR_HPP

#include "boost/program_options.hpp"
#include <algorithm>
#include <boost/algorithm/string.hpp>
#include <cstdlib>
#include <cmath>
#include <cstring>
#include <cctype>
#include <chrono>
#include <ctime>
#include <fstream>
#include <future>
#include <iostream>
#include <locale>
#include <string>
#include <vector>

#include "cacheCluster.hpp"
#include "cacheClusterThread.hpp"
#include "cacheServer.hpp"
#include "constCDNSimulator.hpp"
#include "utils.h"

#include <glib.h>
#include "libCacheSim.h"

namespace CDNSimulator {

    typedef struct {
        std::string param_string;
        std::string trace_path;
        std::string cache_alg;
        std::string exp_name;

        std::string trace_format_str;
        trace_format_e trace_format;
        trace_type_e trace_type;
        obj_id_type_e obj_id_type;
        reader_init_param_t reader_init_params;

        unsigned long n_server;
        unsigned long server_cache_size;
        unsigned long *server_cache_sizes;
        uint32_t *server_weight;

        std::string cluster_mode_str;
        cluster_mode_e cluster_mode;
        double gutter_space;

        std::string log_folder;
        unsigned long log_interval;
        // bool ICP;
        bool check_one_more;
        bool parity_rebalance;

        // bool pseudo_update;
        unsigned int EC_n;
        unsigned int EC_k;
        unsigned int EC_size_threshold;

        unsigned int admission; // only cache an object after being requested admission times
        std::string failure_data; // this should be a file, each line of which state
        // the failed server for the next 5 min
    } simulator_arg_t;

    simulator_arg_t default_sim_arg();

    void parse_cmd_arg(int argc, char **argv, simulator_arg_t *sim_arg);

    void prepare_arg(simulator_arg_t *sim_arg);

    void run(simulator_arg_t *sim_arg);

} // namespace CDNSimulator

#endif /* SIMULATOR_HPP */
