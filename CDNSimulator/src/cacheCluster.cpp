//
//  cacheCluster.cpp
//  CDNSimulator
//
//  Created by Juncheng Yang on 7/13/20.
//  Copyright © 2017 Juncheng. All rights reserved.
//

#include "inttypes.h"
#include <algorithm>
#include <iomanip>
#include <numeric>
#include <unordered_map>
#include <unordered_set>

#include "ortools/graph/max_flow.h"

#include "../libCacheSim/dataStructure/hashtable/hashtable.h"
#include "libCacheSim.h"

#include "C2DNconfig.h"
#include "cacheCluster.hpp"
#include "constCDNSimulator.hpp"

using namespace std;

namespace CDNSimulator {

cacheCluster::cacheCluster(cluster_params_t cluster_params,
                                                     EC_params_t ec_params)
        : hasher(cluster_params.n_server), cluster_params(cluster_params),
            ec_params(ec_params),
            cluster_stat(cluster_params.cluster_id, cluster_params.n_server,
                                     cluster_params.server_cache_sizes, ec_params.n,
                                     ec_params.k) {

    uint64_t i;
#if defined(TRACK_BUCKET_MISS_RATIO)
    bucket_stat.resize(N_BUCKET);
#elif defined(TRACK_SERVER_MISS_RATIO)
    bucket_stat.resize(cluster_params.n_server);
#else
#error not TRACK_BUCKET_MISS_RATIO or TRACK_SERVER_MISS_RATIO
#endif
    read_load_byte.resize(cluster_params.n_server, 0);
    write_load_byte.resize(cluster_params.n_server, 0);

    for (i = 0; i < cluster_params.n_server; i++) {
        this->cache_servers.push_back(cluster_params.cache_servers[i]);
    }

    ring = ch_ring_create(int(cluster_params.n_server),
                                                cluster_params.server_weight);

#ifdef USE_BUCKET_HASHING
    populate_hash_mapping();
#endif

    srand((unsigned int)time(nullptr));
}


void cacheCluster::find_parity_mapping() {
    using namespace operations_research;

    std::vector<int64_t> start_nodes;
    std::vector<int64_t> end_nodes;
    std::vector<int64_t> capacities;

    SimpleMaxFlow max_flow;

    int n_server = n_bucket_per_server.size();

    int server_cap =
            (int)(ceil((double)N_BUCKET * ec_params.n / cluster_stat.n_avail_server));

    vector<unordered_set<int>> buckets_on_server;
    buckets_on_server.resize(n_server);
    for (int i = 0; i < N_BUCKET; i++) {
        uint32_t *data_chunk_mapping = ch_mapping[i];
        for (unsigned int j = 0; j < ec_params.n - 1; j++) {
            buckets_on_server[data_chunk_mapping[j]].insert(i);
        }
    }

    for (int i = 0; i < n_server; i++) {
        int cap = server_cap - buckets_on_server[i].size();
        if (buckets_on_server[i].empty()) {
            assert(cache_servers.at(i)->is_available() == false);
            cap = 0;
        }
        max_flow.AddArcWithCapacity(0, 2 + i, cap < 0 ? 0 : cap);
        for (int bucket_id = 0; bucket_id < N_BUCKET; bucket_id++) {
            if (buckets_on_server[i].find(bucket_id) != buckets_on_server[i].end()) {
                continue;
            }
            max_flow.AddArcWithCapacity(2 + i, 2 + n_server + bucket_id, 1);
        }
    }

    for (int i = 0; i < N_BUCKET; ++i) {
        max_flow.AddArcWithCapacity(2 + n_server + i, 1, 1);
    }

    max_flow.Solve(0, 1);

    //        std::cout << "Max flow: " << max_flow.OptimalFlow() << std::endl;
    //        std::cout << "" << std::endl;
    //        std::cout << "  Arc    Flow / Capacity" << std::endl;

    int find_n_servers = ec_params.n + MAX_N_UNAVAIL;
    for (int i = 0; i < max_flow.NumArcs(); ++i) {
        //                if (max_flow.Tail(i) == 0) {
        //                    std::cout << max_flow.Tail(i) << " -> " <<
        //                    max_flow.Head(i) << "  "
        //                              << max_flow.Flow(i) << "  / " <<
        //                              max_flow.Capacity(i) << std::endl;
        //                }
        if (max_flow.Tail(i) != 0 && max_flow.Head(i) != 1 &&
                max_flow.Flow(i) > 0) {
            int server_id = max_flow.Tail(i) - 2;
            int bucket_id = max_flow.Head(i) - 2 - n_server;
            for (unsigned int j = find_n_servers - 1; j >= ec_params.n; j--)
                ch_mapping[bucket_id][j] = ch_mapping[bucket_id][j - 1];

            ch_mapping[bucket_id][ec_params.n - 1] = server_id;
            n_bucket_per_server[server_id] += 1;
        }
    }
}

/* used in cluster initialization and unavailability change */
void cacheCluster::populate_hash_mapping() {
    /* used to generate hash mapping during cluster initialization */

    uint32_t original_lead_server;
    uint32_t *server_idxs;

    n_bucket_per_server.assign(cluster_params.n_server, 0);
    uint32_t *data_capacity = new uint32_t[cluster_params.n_server];
    std::fill_n(data_capacity, cluster_params.n_server, N_BUCKET * ec_params.n);

    for (uint64_t i = 0; i < cluster_params.n_server; i++) {
        if (!cache_servers[i]->is_available()) {
            data_capacity[i] = 0;
        }
    }

    for (uint64_t bucket_id = 0; bucket_id < N_BUCKET; bucket_id++) {
        int find_n_servers = ec_params.n + MAX_N_UNAVAIL;
        server_idxs = new uint32_t[find_n_servers];
        ch_ring_get_servers((const uint64_t *)(&bucket_id), ring, find_n_servers,
                                                server_idxs, &original_lead_server, data_capacity);
        ch_mapping[bucket_id] = server_idxs;
        ori_lead_mapping[bucket_id] = original_lead_server;

        unsigned int end =
                cluster_params.parity_rebalance ? ec_params.n - 1 : ec_params.n;
        for (unsigned int j = 0; j < end; j++) {
            n_bucket_per_server[server_idxs[j]] += 1;
        }
    }

    INFO("data chunk bucket assignment: # buckets on each server ");
    for (uint32_t i = 0; i < cluster_params.n_server; i++) {
        printf("%u, ", n_bucket_per_server[i]);
    }
    printf("\n");

    if (cluster_params.parity_rebalance) {
        find_parity_mapping();
    }

    INFO("all chunk bucket assignment: # buckets on each server ");
    for (uint32_t i = 0; i < cluster_params.n_server; i++) {
        printf("%u, ", n_bucket_per_server[i]);
    }
    printf("\n");

    // INFO("bucket assignment\n");
    for (uint32_t i = 0; i < N_BUCKET; i++) {
        uint32_t *servers = get_consistent_hash_mapping((uint64_t)i, &original_lead_server);
        // printf("%u: %u %u %u %u %u\n", i, servers[0], servers[1], servers[2],
        //        servers[3], servers[4]);
    }
}

uint32_t *
cacheCluster::get_consistent_hash_mapping(uint64_t id, uint32_t *original_lead_server) {
    auto p = ch_mapping.find(id);
    uint32_t *server_idxs;
    if (p == ch_mapping.end()) {
#if defined(USE_BUCKET_HASHING)
        assert(false);
#endif
        /* get n+1 servers in the case where one more server is checked */
        server_idxs = new uint32_t[MAX_EC_N];

        unsigned long find_n_servers = ec_params.n + MAX_N_UNAVAIL;
        find_n_servers = find_n_servers > cluster_params.n_server
                                                 ? cluster_params.n_server
                                                 : find_n_servers;
        ch_ring_get_servers((const uint64_t *)(&id), ring, find_n_servers,
                                                server_idxs, original_lead_server, NULL);

        // printf("id %llu servers %u %u %u %u %u %u %u\n", id, server_idxs[0],
        //        server_idxs[1], server_idxs[2], server_idxs[3], server_idxs[4],
        //        server_idxs[5], server_idxs[6]);
        if (ch_mapping.size() < 2000) {
            ch_mapping[id] = server_idxs;
            ori_lead_mapping[id] = *original_lead_server;
        }

        VERBOSE("calculate consistent hash mapping for id %" PRIu64
                        ", mapping %u %u %u %u\n",
                        id, server_idxs[0], server_idxs[1], server_idxs[2], server_idxs[3]);
    } else {
        server_idxs = p->second;
        *original_lead_server = ori_lead_mapping.find(id)->second;
    }

    return server_idxs;
}

uint32_t *cacheCluster::_find_mapped_servers(request_t *const req,
                                                                                         uint32_t *lead_server,
                                                                                         uint32_t *second_lead) {
    /* either 0 or 1, used to select one of the two lead servers in replication
     * and coding */
    unsigned int idx_selected;
    uint32_t *server_idxs;
    uint32_t ori_lead;
    // now find the lead server
    if (ec_params.n == 1 || cluster_params.cluster_mode == erft ||
            cluster_params.cluster_mode == two_rep_always_gutter) {
        // always choose a fixed server returned by consistent hash
        idx_selected = 0;
    } else {
        if (cluster_params.cluster_mode == three_rep_always ||
                cluster_params.cluster_mode == C2DN_add_n_three_rep) {
            idx_selected = (unsigned int)rand() % 3;
        } else {
            // select one of the two lead servers that are returned by load balancer
            idx_selected = (unsigned int)rand() % 2;
        }
    }

    if (cluster_params.trace_format == akamai1bWithBucket) {
        assert(req->bucket_id <= N_BUCKET && req->bucket_id >= 0);
        server_idxs = get_consistent_hash_mapping(req->bucket_id, &ori_lead);
    } else {
#if defined(USE_OBJ_HASHING)
        server_idxs = get_consistent_hash_mapping(req->obj_id_int, &ori_lead);
        req->bucket_id = -1;
#elif defined(USE_BUCKET_HASHING)
        req->bucket_id = req->obj_id_int % N_BUCKET;
        server_idxs = get_consistent_hash_mapping(req->bucket_id, &ori_lead);
#else
#error "need to enable obj hashing or bucket hashing"
#endif
    }
    req->original_server = static_cast<int32_t>(ori_lead);

    /* use the first bit to indicate whether the original server is unavailable */
    *lead_server = server_idxs[idx_selected];
    if (cluster_params.cluster_mode == three_rep_always ||
            cluster_params.cluster_mode == C2DN_add_n_three_rep) {
        *second_lead = server_idxs[(idx_selected - rand() % 2 - 1) % 3];
    } else {
        *second_lead = server_idxs[1 - idx_selected];
    }

    cluster_check.req_host = (int)*lead_server;

    VVERBOSE("req %" PRIu64 " size %" PRIu64 " bucket %" PRIu64
                     ", server idx %d %d %d %d\n",
                     req->obj_id_int, req->obj_size, req->bucket_id, server_idxs[0],
                     server_idxs[1], server_idxs[2], server_idxs[3]);

    return server_idxs;
}

void cacheCluster::_cache_hit_full_obj(request_t *const req,
                                                                             uint32_t lead_server,
                                                                             uint32_t *server_idxs) {
    cluster_check.is_hit = true;
    if (cluster_params.cluster_mode == no_replication ||
            cluster_params.cluster_mode == two_rep_popularity) {
        cache_servers.at(lead_server)->get(req, main_cache);
        cache_servers.at(lead_server)->set_obj_type(req, full_obj);
    } else {
        cache_servers.at(server_idxs[0])->get(req, main_cache);
        cache_servers.at(server_idxs[0])->set_obj_type(req, full_obj);
        if (cluster_params.cluster_mode == two_rep_always ||
                cluster_params.cluster_mode == C2DN ||
                cluster_params.cluster_mode == C2DN_add_n) {
            cache_servers.at(server_idxs[1])->get(req, main_cache);
            cache_servers.at(server_idxs[1])->set_obj_type(req, full_obj);
        } else if (cluster_params.cluster_mode == three_rep_always ||
                             cluster_params.cluster_mode == C2DN_add_n_three_rep) {
            cache_servers.at(server_idxs[1])->get(req, main_cache);
            cache_servers.at(server_idxs[1])->set_obj_type(req, full_obj);
            cache_servers.at(server_idxs[2])->get(req, main_cache);
            cache_servers.at(server_idxs[2])->set_obj_type(req, full_obj);
        } else if (cluster_params.cluster_mode == two_rep_always_gutter) {
            //        if (get_freq(req, false) >= 4) {
            if (rand() % 100 < cache_servers.at(server_idxs[0])->gutter_prob) {
                cache_servers.at(server_idxs[1])->get(req, gutter_cache);
                cache_servers.at(server_idxs[1])->set_obj_type(req, full_obj);
            }
        } else if (cluster_params.cluster_mode == erft) {
            cache_servers.at(server_idxs[1])->get(req, gutter_cache);
            cache_servers.at(server_idxs[1])->set_obj_type(req, full_obj);
        } else {
            abort();
        }
    }
}

/* get chunk info of the chunk servers, return the number of available chunks */
void cacheCluster::_get_chunk_info(request_t *const req, uint32_t *server_idxs,
                                                                     server_chunk_stat_e *server_chunk_status,
                                                                     int *chunk_cnt, bool *checked_servers,
                                                                     unsigned int *n_chunk_hit,
                                                                     unsigned int *n_duplicate_chunks,
                                                                     bool _debug_check) {

    for (unsigned int i = 0; i < ec_params.n; i++) {
        unsigned int server_idx = server_idxs[i];
        if (!_debug_check) {
            read_load_byte[server_idx] += req->obj_size;
            assert(checked_servers[server_idx] == false);
        }
        checked_servers[server_idx] = true;

        obj_type_e obj_type = invalid_obj;
        if (cache_servers.at(server_idx)->check(req, &obj_type)) {
            unsigned int chunk_id = cache_servers.at(server_idx)->get_chunk_id(req);
            if (chunk_cnt[chunk_id] == 0) {
                (*n_chunk_hit)++;
                server_chunk_status[i] = correct_chunk;
            } else {
                (*n_duplicate_chunks)++;
                server_chunk_status[i] = incorrect_chunk;
            }
            chunk_cnt[chunk_id]++;
        } else {
            if (!_debug_check) {
                write_load_byte[server_idx] += req->obj_size;
            }
            server_chunk_status[i] = no_chunk;
        }
    }
}

void cacheCluster::_cache_hit_chunk_obj(request_t *const req,
                                                                                uint32_t lead_server,
                                                                                uint32_t *server_idxs) {
    /* 1 correct chunk, 2 incorrect chunk, 3 has_not_checked 4. invalid */
    static server_chunk_stat_e server_chunk_status[MAX_EC_N];
    static int chunk_cnt[MAX_EC_N];
    static bool checked_servers[MAX_N_SERVER];

    unsigned int n_duplicate_chunks = 0;

    std::fill_n(server_chunk_status, MAX_EC_N, has_not_checked);
    std::fill_n(chunk_cnt, MAX_EC_N, 0);
    std::fill_n(checked_servers, cluster_params.n_server, false);

    uint64_t full_obj_size = req->obj_size;
    uint64_t chunk_obj_size =
            (uint64_t)(ceil(double(req->obj_size) / ec_params.k));
    req->obj_size = chunk_obj_size;

    _get_chunk_info(req, server_idxs, server_chunk_status, chunk_cnt,
                                    checked_servers, &cluster_check.n_chunk_hit,
                                    &n_duplicate_chunks, false);

    if (cluster_check.n_chunk_hit < ec_params.k) {
        if (cluster_params.check_one_more || cluster_params.parity_rebalance) {
            for (unsigned int i = ec_params.n; i < ec_params.n + MAX_N_UNAVAIL; i++) {
                if (checked_servers[server_idxs[i]]) {
                    continue;
                } else {
                    checked_servers[server_idxs[i]] = true;
                }
                obj_type_e obj_type;
                if (cache_servers.at(server_idxs[i])->check(req, &obj_type)) {
                    read_load_byte[server_idxs[i]] += req->obj_size;

                    assert(obj_type == chunk_obj);
                    unsigned int chunk_id =
                            cache_servers.at(server_idxs[i])->get_chunk_id(req);
                    if (chunk_cnt[chunk_id] == 0) {
                        cluster_check.n_chunk_hit++;
                    }
                }
            }
        }
        /* check prev rebalanced mapping */
        if (cluster_check.n_chunk_hit < ec_params.k &&
                cluster_params.parity_rebalance &&
                prev_ch_mapping.count(req->bucket_id) > 0) {
            uint32_t *prev_server_idxs = prev_ch_mapping[req->bucket_id];
            // for (unsigned int i = 0; i < ec_params.n; i++) {
            //   unsigned int server_idx = prev_server_idxs[i];
            unsigned int server_idx = prev_server_idxs[ec_params.n - 1];
            if (!checked_servers[server_idx]) {
                checked_servers[server_idx] = true;
                obj_type_e obj_type;
                if (cache_servers.at(server_idx)->check(req, &obj_type)) {
                    read_load_byte[server_idx] += req->obj_size;

                    assert(obj_type == chunk_obj);
                    unsigned int chunk_id =
                            cache_servers.at(server_idx)->get_chunk_id(req);
                    if (chunk_cnt[chunk_id] == 0) {
                        cluster_check.n_chunk_hit++;
                    }
                }
            }
            // }
        }
    }

    if (cluster_check.n_chunk_hit < ec_params.n &&
            n_duplicate_chunks > ec_params.n - ec_params.k) {
        std::cerr << "obj " << req->obj_id_int << ": find too many ("
                            << n_duplicate_chunks << ") duplicate chunks, "
                            << cluster_check.n_chunk_hit << " chunk is_hit, ";
        std::cerr << "chunk cnt: " << chunk_cnt[0] << "," << chunk_cnt[1] << ","
                            << chunk_cnt[2] << "," << chunk_cnt[3] << ", ";
        std::cerr << "server(chunk_id) ";
        for (unsigned int i = 0; i < cache_servers.size(); i++) {
            std::cerr << (uint32_t)i;
            obj_type_e obj_type;
            if (!cache_servers.at(i)->check(req, &obj_type))
                std::cerr << "(" << -1 << ") ";
            else
                std::cerr << "(" << (uint32_t)cache_servers.at(i)->get_chunk_id(req)
                                    << ") ";
        }
        std::cerr << std::endl;
    }

    // restore all chunks
    //      for (i = 0; i < ec_params.n; i++) {
    //        cache_servers.at(server_idxs[i])->get(req, unknown_cache);
    //        cache_servers.at(server_idxs[i])->set_chunk_id(req, i);
    //      }

    /* restore missing chunk only */
    unsigned int i = 0;
    for (unsigned int j = 0; j < ec_params.n; j++) {
        if (chunk_cnt[j] == 0) {
            //          std::cerr << "missing " << j << std::endl;
            while (server_chunk_status[i] == correct_chunk)
                i++;
            assert(i < ec_params.n);
            cache_servers.at(server_idxs[i])->get(req, unknown_cache);
            cache_servers.at(server_idxs[i])->set_chunk_id(req, j);
            //          if (i != j) {
            //      std::cerr << "set " << i << " to " << j << std::endl;
            //            print = true;
            //          }
            i++;
        }
    }

    cluster_stat.intra_cluster_bytes += n_duplicate_chunks * chunk_obj_size;

    if (cluster_check.n_chunk_hit < ec_params.k) {
        cluster_stat.midgress_bytes +=
                (ec_params.k - cluster_check.n_chunk_hit) * chunk_obj_size;
#if defined(TRACK_BUCKET_MISS_RATIO)
        bucket_stat[req->bucket_id].midgress +=
                (ec_params.k - cluster_check.n_chunk_hit) * chunk_obj_size;
#elif defined(TRACK_SERVER_MISS_RATIO)
        bucket_stat[req->original_server].midgress +=
                (ec_params.k - cluster_check.n_chunk_hit) * chunk_obj_size;
#endif
    } else {
        cluster_check.is_hit = true;
    }
    cluster_stat.intra_cluster_bytes += chunk_obj_size * (ec_params.n - 1);
    cluster_stat.n_miss[ec_params.n - cluster_check.n_chunk_hit]++;

    req->obj_size = full_obj_size;
}

void cacheCluster::_cache_miss_chunk_obj(request_t *req, uint32_t *server_idxs) {
    // code object
    uint64_t full_obj_size = req->obj_size;
    uint64_t chunk_obj_size =
            (uint64_t)(ceil(double(full_obj_size) / ec_params.k));
    req->obj_size = chunk_obj_size;
    cluster_stat.intra_cluster_bytes += chunk_obj_size * (ec_params.k - 1);
    for (unsigned int i = 0; i < ec_params.k; i++) {
        write_load_byte[server_idxs[i]] += req->obj_size;

        cache_servers.at(server_idxs[i])->get(req, main_cache);
        cache_servers.at(server_idxs[i])->set_chunk_id(req, i);
    }
    if (cluster_params.cluster_mode == C2DN_add_n ||
            cluster_params.cluster_mode == C2DN_add_n_three_rep) {
        cluster_stat.intra_cluster_bytes += chunk_obj_size * (ec_params.n - 1);
        for (unsigned int i = ec_params.k; i < ec_params.n; i++) {
            write_load_byte[server_idxs[i]] += req->obj_size;

#ifdef USE_GUTTER_SPACE
            cache_servers.at(server_idxs[i])->get(req, gutter_cache);
#else
            cache_servers.at(server_idxs[i])->get(req, main_cache);
#endif
            cache_servers.at(server_idxs[i])->set_chunk_id(req, i);
        }
    }
    req->obj_size = full_obj_size;
}

void cacheCluster::_cache_miss_full_obj(request_t *req, uint32_t lead_server,
                                                                                uint32_t *server_idxs) {
    if (cluster_params.cluster_mode == no_replication ||
            cluster_params.cluster_mode == two_rep_popularity) {
        write_load_byte[lead_server] += req->obj_size;
        cache_servers.at(lead_server)->get(req, main_cache);
        cache_servers.at(lead_server)->set_obj_type(req, full_obj);

    } else if (cluster_params.cluster_mode == two_rep_always ||
                         cluster_params.cluster_mode == C2DN ||
                         cluster_params.cluster_mode == C2DN_add_n) {
        for (int i = 0; i < 2; i++) {
            write_load_byte[server_idxs[i]] += req->obj_size;
            cache_servers.at(server_idxs[i])->get(req, main_cache);
            cache_servers.at(server_idxs[i])->set_obj_type(req, full_obj);
        }
    } else if (cluster_params.cluster_mode == three_rep_always ||
                         cluster_params.cluster_mode == C2DN_add_n_three_rep) {
        for (int i = 0; i < 3; i++) {
            write_load_byte[server_idxs[i]] += req->obj_size;
            cache_servers.at(server_idxs[i])->get(req, main_cache);
            cache_servers.at(server_idxs[i])->set_obj_type(req, full_obj);
        }
    } else if (cluster_params.cluster_mode == two_rep_always_gutter) {
        for (int i = 0; i < 2; i++) {
            write_load_byte[server_idxs[i]] += req->obj_size;
            cache_servers.at(server_idxs[i])->get(req, main_cache);
            cache_servers.at(server_idxs[i])->set_obj_type(req, full_obj);
        }
    } else {
        std::cerr << "unknown_obj_type cluster mode " << cluster_params.cluster_mode
                            << std::endl;
        abort();
    }
}

cluster_hit_t cacheCluster::get(request_t *const req) {
    cluster_check = {false, false, false, 0, unknown_obj_type};
    // printf("%d\n", req->obj_size); }

    /** first get the index of the cacheServer which
     *  this request will be sent to using consistent hashing */
    uint32_t lead_server, second_lead;
    uint32_t *server_idxs = _find_mapped_servers(req, &lead_server, &second_lead);

    cluster_stat.cluster_req_cnt++;
    cluster_stat.cluster_req_bytes += req->obj_size;

#if defined(TRACK_BUCKET_MISS_RATIO)
    (bucket_stat[req->bucket_id].req_cnt)++;
    bucket_stat[req->bucket_id].req_byte += req->obj_size;
#elif defined(TRACK_SERVER_MISS_RATIO)
    (bucket_stat[req->original_server].req_cnt)++;
    bucket_stat[req->original_server].req_byte += req->obj_size;
#endif

    /* found a chunk or full object in the lead server */
    bool found_in_lead_server =
            cache_servers.at(lead_server)->check(req, &cluster_check.obj_type);
    /* found in other servers via ICP */
    // bool found_via_ICP = false;
    /* whether admit into cache if miss */
    bool admit_into_cache = _should_admit(req);
    int server_to_read;

    if (found_in_lead_server) {
        server_to_read = lead_server;
    } else {
        if (cluster_params.check_one_more) {
            found_in_lead_server =
                    cache_servers.at(second_lead)->check(req, &cluster_check.obj_type);
        }
        if (found_in_lead_server) {
            server_to_read = second_lead;
        }
    }

    /* we have checked lead server and possibly other servers if ICP enabled,
     * if it is a miss, the full object will be fetched from origin */
    // if (found_in_lead_server || found_via_ICP) {
    if (found_in_lead_server) {
        // cache is_hit, check whether it is full object
        if (cluster_check.obj_type == full_obj) {
            read_load_byte[server_to_read] += req->obj_size;
            _cache_hit_full_obj(req, lead_server, server_idxs);

        } else if (cluster_check.obj_type == chunk_obj) {
            // we find a chunk, now get chunks from the peer caches
            _cache_hit_chunk_obj(req, lead_server, server_idxs);

        } else if (cluster_check.obj_type == unknown_obj_type) {
            ERROR("found obj %" PRId64
                        " in cache, but does not find obj_type information, "
                        "found in lead server %d\n",
                        req->obj_id_int, found_in_lead_server);
            abort();
        }
    } else {
        // object miss, fetch full object from origin and save to n caches
        cluster_stat.midgress_bytes += req->obj_size;
#if defined(TRACK_BUCKET_MISS_RATIO)
        bucket_stat[req->bucket_id].midgress += req->obj_size;
#elif defined(TRACK_SERVER_MISS_RATIO)
        bucket_stat[req->original_server].midgress += req->obj_size;
#endif
        if (admit_into_cache) {
            read_load_byte[lead_server] += req->obj_size;
            // this object is not one(n)-is_hit-wonder
            if (check_coding_policy(req)) {
                _cache_miss_chunk_obj(req, server_idxs);
            } else {
                // full object
                _cache_miss_full_obj(req, lead_server, server_idxs);
            }
        }
    }


    if (cluster_check.is_hit) {
        // assert(cluster_check.n_chunk_hit >= ec_params.k); 
        assert(cluster_check.n_chunk_hit <= ec_params.n); 

        cluster_stat.cluster_hit_cnt++;
        cluster_stat.cluster_hit_bytes += req->obj_size;
#if defined(TRACK_BUCKET_MISS_RATIO)
        (bucket_stat[req->bucket_id].hit_cnt)++;
#elif defined(TRACK_SERVER_MISS_RATIO)
        (bucket_stat[req->original_server].hit_cnt)++;
#endif
        if (cluster_check.obj_type == full_obj) {
            cluster_stat.full_obj_hit_cnt += 1;
            cluster_stat.full_obj_hit_bytes += req->obj_size;
        } else if (cluster_check.obj_type == chunk_obj) {
            cluster_stat.chunk_obj_hit_cnt += 1;
            cluster_stat.chunk_obj_hit_bytes += req->obj_size;
        } else {
            ERROR("unknown_obj_type %d\n", cluster_check.obj_type);
            abort();
        }
    } 

    return cluster_check;
}

void _objmap_aux(cache_obj_t *cache_obj, void *user_data) {
    auto cluster_objmap =
            (std::unordered_map<uint64_t, std::pair<uint64_t, int8_t>> *)user_data;
    int8_t cnt = 1;
    if (cluster_objmap->find(cache_obj->obj_id_int) != cluster_objmap->end()) {
        cnt = (*cluster_objmap)[cache_obj->obj_id_int].second + 1;
    }
    (*cluster_objmap)[cache_obj->obj_id_int] =
            std::make_pair(cache_obj->obj_size, cnt);
}

/**
 * find the replication factor in 2-rep
 * @param obj replication coefficient in terms of obj
 * @param byte replication coefficient in terms of bytes
 */
void cacheCluster::find_rep_factor(double *obj_rep, double *byte_rep,
                                                                     uint64_t *n_chunk_cnt,
                                                                     uint64_t *n_server_obj,
                                                                     uint64_t *n_server_byte,
                                                                     uint64_t *n_cluster_obj,
                                                                     uint64_t *n_cluster_byte) {
    /* first create a giant map to store all the cached objects in the cluster */

    // the sum of each server in the cluster
    uint64_t total_stored_size_servers = 0;
    uint64_t n_total_stored_obj_servers = 0;

    // for the cluster (de-duplication version of the ones above)
    uint64_t total_stored_size_cluster = 0;
    uint64_t n_total_stored_obj_cluster = 0;

    std::unordered_map<uint64_t, std::pair<uint64_t, int8_t>> cluster_objmap;

    for (unsigned int i = 0; i < get_num_server(); i++) {
        if (!cache_servers.at(i)->is_available())
            continue;
        n_total_stored_obj_servers += cache_servers.at(i)->cache->n_obj;
        total_stored_size_servers += cache_servers.at(i)->cache->occupied_size;
        hashtable_foreach(cache_servers.at(i)->cache->hashtable, _objmap_aux,
                                            (void *)&cluster_objmap);
    }

    n_total_stored_obj_cluster = cluster_objmap.size();
    memset(n_chunk_cnt, 0, sizeof(uint64_t) * ec_params.n);
    for (auto kv : cluster_objmap) {
        total_stored_size_cluster += kv.second.first;
        n_chunk_cnt[kv.second.second - 1] += 1;
    }

    *n_server_obj = n_total_stored_obj_servers;
    *n_server_byte = total_stored_size_servers;
    *n_cluster_obj = n_total_stored_obj_cluster;
    *n_cluster_byte = total_stored_size_cluster;

    *obj_rep =
            (double)n_total_stored_obj_servers / (double)n_total_stored_obj_cluster;
    *byte_rep =
            (double)total_stored_size_servers / (double)total_stored_size_cluster;
}

cacheCluster::~cacheCluster() {
    reset_consistent_hash_mapping();
    ch_ring_destroy(ring);
    if (curr_parity_ring)
        ch_ring_destroy(curr_parity_ring);
}
} // namespace CDNSimulator
