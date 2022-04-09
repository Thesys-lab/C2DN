//
//  cacheClusterThread.cpp
//  CDNSimulator
//
//  Created by Juncheng Yang on 11/20/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#include "../libCacheSim/dataStructure/hash/hash.h"
#include "libCacheSim.h"
#include "cacheClusterThread.hpp"
#include "include/C2DNconfig.h"
#include <exception>
#include <string>

using namespace std;
using namespace CDNSimulator;

void cacheClusterThread::output_rep_coef(ofstream &ofs, bool print) {
    uint64_t n_server_obj, n_server_byte, n_cluster_obj, n_cluster_byte;
    std::stringstream ss;
    double rep_coef_obj, rep_coef_byte;
    uint64_t n_chunk_cnt[cache_cluster->ec_params.n];

    cache_cluster->find_rep_factor(&rep_coef_obj, &rep_coef_byte, n_chunk_cnt,
                                   &n_server_obj, &n_server_byte, &n_cluster_obj, &n_cluster_byte);
    if (cluster_update_interval != log_interval) {
        WARNING("cluster update interval (%lu) != log interval (%lu)\n", cluster_update_interval, log_interval);
    }

    if (print) {
        ss << "vtime, rep coefficient obj/byte, n_server_obj/n_cluster_obj, n_server_byte/n_cluster_byte" << std::endl;
        INFO("%s", ss.str().c_str());
        ofs << ss.str();
        ss.str("");

        ss << vtime << ", " << rep_coef_obj << "/" << rep_coef_byte
           << ", " << n_server_obj << "/" << n_cluster_obj << ", " << n_server_byte << "/"
           << n_cluster_byte << std::endl;
        INFO("%s", ss.str().c_str());
        ofs << ss.str();
        ss.str("");
    }

    ss << "n_chunk_cnt (1-" << cache_cluster->ec_params.n << ") ";
    for (unsigned int i = 0; i < cache_cluster->ec_params.n; i++)
        ss << n_chunk_cnt[i] << ",";
    ss << std::endl;
    if (print)
        INFO("%s", ss.str().c_str());
    ofs << ss.str();
}

void cacheClusterThread::run() {
    request_t *req = new_request();
    log_ofstream << "#" << this->param_string << std::endl;
    log_ofstream << "#vtime, " << cacheClusterStat::stat_str_header(false);

    read_one_req(reader, req);
    if (req->obj_size <= 0) {
        std::cerr << "request size 0" << std::endl;
        req->obj_size = 100 * 1024;
    }

    cache_start_time = req->real_time;
    last_cluster_unavailability_update = req->real_time;
    last_log_otime = req->real_time;
    last_rep_coef_log_otime = req->real_time;
    cluster_hit_t hit_result;
    INFO("cache cluster started, trace start time %ld\n", cache_start_time);


    std::vector<int> failed_servers, prev_failed_servers(0);

    while (req->valid) {
        if (!failure_vector_deque.empty()) {
            // update server availability info
#ifdef USE_VTIME
            if (vtime - last_cluster_unavailability_update >= cluster_update_interval) {
#else
            if (req->real_time - last_cluster_unavailability_update >= cluster_update_interval) {
#endif
                failed_servers = failure_vector_deque.front();
                if (failed_servers != prev_failed_servers) {
                    for (unsigned long i = 0; i < cache_cluster->get_num_server(); i++) {
                        cache_cluster->recover_one_server(i);
                    }
                    for (auto &it: failed_servers) {
                        if (cache_cluster->fail_one_server(it) != 0) {
                            std::cerr << "fail to create server failure" << std::endl;
                        }
                    }
#ifdef USE_BUCKET_HASHING
                    cache_cluster->populate_hash_mapping();
#endif
                }
                prev_failed_servers = failure_vector_deque.front();
                failure_vector_deque.pop_front();
#ifdef USE_VTIME
                last_cluster_unavailability_update = vtime;
#else
                last_cluster_unavailability_update = req->real_time;
#endif
            }
        }

        vtime++;
        hit_result = cache_cluster->get(req);

#ifdef TRACK_REQ_CORR_OVER_TIME
        n_chunk_hit[hit_result.n_chunk_hit] += 1;
        if (log_interval != 0 && req->real_time - last_log_otime >= log_interval) {
            for (uint i = 0; i < cache_cluster->ec_params.n + 1; i++) {
                corr_ofstream << n_chunk_hit[i] << " ";
                n_chunk_hit[i] = 0;
            }
            corr_ofstream << endl;
        }
#endif


#ifdef TRACK_OBJ_CORR_OVER_TIME
        if (log_interval != 0 && req->real_time - last_rep_coef_log_otime >= log_interval * 6) {
            rep_coef_ofstream << req->real_time << " "; 
            output_rep_coef(rep_coef_ofstream, false);
            last_rep_coef_log_otime = req->real_time;
        }
#endif

#ifdef USE_VTIME
        if (log_interval != 0 && vtime - last_log_otime >= log_interval) {
#else
        if (log_interval != 0 && req->real_time - last_log_otime >= log_interval) {
#endif
            last_log_otime = req->real_time;
            log_ofstream << req->real_time << ", " << cluster_stat->stat_str(false, false);

            /* bucket stat */
#if defined(TRACK_BUCKET_MISS_RATIO)
            for (size_t i = 0; i < N_BUCKET; i++) {
#elif defined(TRACK_SERVER_MISS_RATIO)
            for (size_t i = 0; i < cache_cluster->get_num_server(); i++) {
#endif
                log_bucket_ofstream << cache_cluster->bucket_stat[i].stat_str() << ",";
                cache_cluster->bucket_stat[i].clear();
            }
            log_bucket_ofstream << std::endl;
        }


        read_one_req(reader, req);
    }


#ifdef FIND_REP_COEF
    output_rep_coef(log_ofstream, true);
#endif


    free_request(req);

    std::stringstream ss;
    ss << "read load ";
    for (unsigned int i=0; i<cache_cluster->get_num_server(); i++)
        ss << cache_cluster->read_load_byte[i] << ", ";
    ss << std::endl;
    INFO("%s", ss.str().c_str());
    log_ofstream << ss.str();
    ss.str("");

    ss << "write load ";
    for (unsigned int i=0; i<cache_cluster->get_num_server(); i++)
        ss << cache_cluster->write_load_byte[i] << ", ";
    ss << std::endl;
    INFO("%s", ss.str().c_str());
    log_ofstream << ss.str();

    INFO("%s", cluster_stat->final_stat_header().c_str());
    INFO("%s", cluster_stat->final_stat().c_str());
    log_ofstream << cluster_stat->final_stat_header() << std::endl;
    log_ofstream << cluster_stat->final_stat() << std::endl;
}
//} // namespace CDNSimulator
