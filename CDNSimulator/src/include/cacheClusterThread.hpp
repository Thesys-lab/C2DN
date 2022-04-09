//
//  cacheClusterThread.hpp
//  CDNSimulator
//
//  Created by Juncheng Yang on 11/20/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#ifndef CACHE_CLUSTER_THREAD_hpp
#define CACHE_CLUSTER_THREAD_hpp

#include <algorithm>
#include <atomic>
#include <chrono>
#include <climits>
#include <condition_variable>
#include <cstdlib>
#include <ctime>
#include <deque>
#include <fstream>
#include <functional>
#include <future>
#include <iostream>
#include <mutex>
#include <queue>
#include <string>
#include <thread>
#include <vector>
#include <boost/algorithm/string.hpp>

#include <glib.h>
#include <stdio.h>
#include <sys/stat.h>
#include <sys/types.h>

#include "cacheCluster.hpp"
#include "constCDNSimulator.hpp"
#include "config.h"

using namespace std;

namespace CDNSimulator {

    class cp_comparator {
    public:
        bool operator()(request_t *a, request_t *b) {
            return (a->real_time > b->real_time);
        }
    };

    class cacheClusterThread {

        string param_string;
        string exp_name;
        reader_t *reader;

        string failure_data_file;
        deque<vector<int>> failure_vector_deque;

        cacheCluster *cache_cluster;
        cacheClusterStat *cluster_stat;

        unsigned long cache_start_time = 0;
        unsigned long last_cluster_unavailability_update;
#ifdef USE_VTIME
        unsigned long cluster_update_interval = 100 * 1000;
#else
        unsigned long cluster_update_interval = 5 * 60;
#endif
        unsigned long vtime;
        unsigned long last_log_otime;
        unsigned long last_rep_coef_log_otime;
        unsigned long log_interval;

        string log_folder;
        string ofilename;
        ofstream log_ofstream;
        ofstream log_bucket_ofstream;

#ifdef TRACK_OBJ_CORR_OVER_TIME
        ofstream rep_coef_ofstream;
#endif

#ifdef TRACK_REQ_CORR_OVER_TIME
        ofstream corr_ofstream;
        uint64_t n_chunk_hit[128] = {0};
#endif


    public:
        cacheClusterThread(string param_string,
                           string exp_name,
                           cacheCluster *cache_cluster,
                           reader_t *const reader,
                           string failure_data_file,
                           string log_folder,
                           unsigned long log_interval = 20000)
                : param_string(move(param_string)),
                  exp_name(exp_name),
                  reader(reader),
                  failure_data_file(failure_data_file),
                  cache_cluster(cache_cluster),
                  cluster_stat(&cache_cluster->cluster_stat), vtime(0),
                  last_log_otime(0), log_interval(log_interval), log_folder(move(log_folder)),
                  log_ofstream() {

            ofilename = this->log_folder + "/cluster" + exp_name;
            log_ofstream.open(ofilename, ofstream::out | ofstream::trunc);
            log_bucket_ofstream.open(ofilename + ".bucket", ofstream::out | ofstream::trunc);

            load_failure_data();

#ifdef TRACK_OBJ_CORR_OVER_TIME
            rep_coef_ofstream.open(ofilename + ".obj_n_chunk", ofstream::out | ofstream::trunc);
            rep_coef_ofstream.precision(4);
            rep_coef_ofstream
                    << "vtime, rep coefficient obj/byte, n_server_obj/n_cluster_obj, n_server_byte/n_cluster_byte"
                    << endl;
#endif


#ifdef TRACK_REQ_CORR_OVER_TIME
            corr_ofstream.open(ofilename + ".req_n_chunk", ofstream::out | ofstream::trunc);
#endif

        };

        void load_failure_data() {
            if (!failure_data_file.empty()) {
                string line;
                ifstream failure_data_ifstream;
                failure_data_ifstream.open(failure_data_file);
                while (getline(failure_data_ifstream, line)) {
                    vector<string> line_split;
                    vector<int> failed_servers;

                    if (!line.empty()) {
                        boost::split(line_split, line, [](char c) { return c == ' '; });
                        for (auto &it: line_split) {
        //  cout << it << endl;
                            failed_servers.push_back(stoi(it));
                        }
                    }
                    failure_vector_deque.push_back(failed_servers);
//        cout << ", " << failed_servers.size() << ", " << failure_vector_deque.size() << endl;
                }
                failure_data_ifstream.close();
                INFO("load failure data %lu pts\n", failure_vector_deque.size());
            }
        }

        void run();

        void output_rep_coef(ofstream &ofs, bool);

        cacheCluster *get_cache_cluster() { return this->cache_cluster; }

        cacheClusterStat *get_cluster_stat() { return this->cluster_stat; }

        ~cacheClusterThread() {
#ifdef TRACK_OBJ_CORR_OVER_TIME
            rep_coef_ofstream.close();
#endif
#ifdef TRACK_REQ_CORR_OVER_TIME
            corr_ofstream.close();
#endif

            log_bucket_ofstream.close();
            log_ofstream.close();
        }
    };

} // namespace CDNSimulator

#endif /* CACHE_CLUSTER_THREAD_hpp */
