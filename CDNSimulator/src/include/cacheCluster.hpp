//
//  cacheCluster.hpp
//  CDNSimulator
//
//  Created by Juncheng Yang on 11/18/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#ifndef CACHE_CLUSTER_HPP
#define CACHE_CLUSTER_HPP

#include <glib.h>
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "libCacheSim.h"

#include "cacheServer.hpp"
#include "consistentHash.hpp"
#include "constCDNSimulator.hpp"
#include "hasher.hpp"
#include "stat.hpp"
#include <exception>
#include <fstream>
#include <iostream>
#include <map>
#include <ostream>
#include <random>
#include <sstream>
#include <stdexcept>
#include <string>
#include <unordered_map>
#include <vector>

#include "C2DNconfig.h"

#undef RECORD_SERVER_STAT

namespace CDNSimulator {

typedef struct {
  bool is_hit;
  bool is_ICP_hit;
  bool is_RAM_hit;
  unsigned int n_chunk_hit;
  int req_host;
  obj_type_e obj_type;
} cluster_hit_t;

typedef enum {
  no_replication = 1,
  two_rep_popularity = 2,
  two_rep_always,
  two_rep_always_gutter,
  C2DN,
  C2DN_add_n,

  three_rep_always,
  C2DN_add_n_three_rep,

  erft,

  invalid_mode
} cluster_mode_e;

typedef struct {
  unsigned long cluster_id;
  std::string exp_name;
  trace_format_e trace_format;
  cacheServer **cache_servers;
  unsigned long n_server;
  unsigned long *server_cache_sizes;
  uint32_t *server_weight;
  unsigned int admission;
  // bool ICP;
  bool check_one_more;
  bool parity_rebalance;

  cluster_mode_e cluster_mode;
  double gutter_space;
} cluster_params_t;

typedef struct {
  unsigned int n;
  unsigned int k;
  unsigned int size_threshold;
  // bool pseudo_req;
} EC_params_t;

typedef enum {
  correct_chunk = 1,
  incorrect_chunk,
  no_chunk,
  has_not_checked,

  invalid_chunk_status
} server_chunk_stat_e;

class cacheCluster {
private:
  myHasher hasher;
  ring_t *ring;
  ring_t *curr_parity_ring = NULL;

  std::unordered_map<uint32_t, uint32_t *>
      ch_mapping; // mapping from obj_id or bucket_id to consistent hash
  std::unordered_map<uint32_t, uint32_t *>
      prev_ch_mapping; // mapping from obj_id or bucket_id to consistent hash
  std::unordered_map<uint32_t, uint32_t>
      ori_lead_mapping; // mapping from obj_id or bucket_id to consistent hash
  std::vector<uint32_t>
      n_bucket_per_server; /* number of (chunk) buckets on each server */

  std::unordered_map<uint32_t, unsigned int> obj_freq_map;

  cluster_hit_t cluster_check = {false, false, false, 0};

  uint32_t *_find_mapped_servers(request_t *const req, uint32_t *lead_server,
                                 uint32_t *second_lead);

  void _cache_hit_full_obj(request_t *const req, uint32_t lead_server,
                           uint32_t *server_idxs);

  void _get_chunk_info(request_t *const req, uint32_t *server_idxs,
                      server_chunk_stat_e *server_chunk_status, int *chunk_cnt,
                      bool *checked_servers, unsigned int *n_chunk_hit,
                      unsigned int *n_duplicate_chunks, bool _debug_check);

  void _cache_hit_chunk_obj(request_t *const req, uint32_t lead_server,
                            uint32_t *server_idxs);

  void _cache_miss_chunk_obj(request_t *req, uint32_t *server_idxs);

  void _cache_miss_full_obj(request_t *req, uint32_t lead_server,
                            uint32_t *server_idxs);

  inline bool _should_admit(request_t *const req) {
    /* whether admitting current request into cache if it is a miss */
    bool admit_into_cache = false;
    if (cluster_params.admission <= 0) {
      admit_into_cache = true;
    } else {
      if (get_freq(req, false) >= cluster_params.admission) {
        admit_into_cache = true;
      }
      get_freq(req, true);
    }
    return admit_into_cache;
  }

  void find_parity_mapping();

  uint32_t *get_consistent_hash_mapping(uint64_t id,
                                        uint32_t *original_lead_server);

  inline bool check_coding_policy(request_t *req) {
    if (cluster_params.cluster_mode == no_replication ||
        cluster_params.cluster_mode == two_rep_popularity ||
        cluster_params.cluster_mode == two_rep_always ||
        cluster_params.cluster_mode == two_rep_always_gutter ||
        cluster_params.cluster_mode == three_rep_always)
      return false;
    else if (cluster_params.cluster_mode == C2DN ||
             cluster_params.cluster_mode == C2DN_add_n ||
             cluster_params.cluster_mode == C2DN_add_n_three_rep) {
      if (req->obj_size > ec_params.size_threshold) {
        return true;
      } else {
        return false;
      }
    } else {
      abort();
    }
  }

public:
  cluster_params_t cluster_params;
  EC_params_t ec_params;
  std::vector<uint64_t> read_load_byte;
  std::vector<uint64_t> write_load_byte;

  std::vector<cacheServer *> cache_servers;
  cacheClusterStat cluster_stat;
  std::vector<bucketStat> bucket_stat;

  cacheCluster(cluster_params_t cluster_params, EC_params_t ec_params);

  /** this is used to add request to current cluster **/
  cluster_hit_t get(request_t *req);

  inline size_t get_num_server() { return this->cache_servers.size(); };

  //        void _create_parity_consistent_hash_ring();

  inline unsigned int get_freq(request_t *req, bool inc) {
    uint32_t key = req->obj_id_int;
    unsigned int freq = 0;
    auto p = obj_freq_map.find(key);
    if (p != obj_freq_map.end()) {
      freq = p->second;
      if (inc)
        p->second = freq + 1;
    } else {
      if (inc)
        obj_freq_map[key] = freq + 1;
    }

    return freq;
  }

  void populate_hash_mapping();

  inline void print_consistent_hash_mapping() {
    for (auto &kv : ch_mapping) {
      std::cout << kv.first << ": ";
      for (int x = 0; x < 10; x++)
        std::cout << (uint64_t)(kv.second[x]) << "\t";
      std::cout << std::endl;
    }
  }

  inline void reset_consistent_hash_mapping() {
    DEBUG("reset consistent hash mapping\n");
    if (!ch_mapping.empty() && !prev_ch_mapping.empty()) {
      auto e = prev_ch_mapping.begin();
      while (e != prev_ch_mapping.end()) {
        delete[]((uint32_t *)e->second);
        prev_ch_mapping.erase((e++)->first);
      }
    }
    prev_ch_mapping = ch_mapping;
    ch_mapping.clear();
    assert(ch_mapping.empty()); 
  }

  inline bool fail_one_server(unsigned int server_id) {
    INFO("server %u fails\n", server_id);
    reset_consistent_hash_mapping();
    cache_servers.at(server_id)->set_unavailable();
    cluster_stat.n_avail_server--;
    return cache_servers.at(server_id)->is_available();
  };

  inline bool recover_one_server(unsigned int server_id) {
    if (!cache_servers.at(server_id)->is_available()) {
      INFO("recover server %u\n", server_id);
      reset_consistent_hash_mapping();
      cache_servers.at(server_id)->set_available();
      cluster_stat.n_avail_server++;
    }
    return cache_servers.at(server_id)->is_available();
  };

  inline int get_num_copy_in_cluster(request_t *req) {
    int n_copy = 0;
    obj_type_e obj_type;
    for (size_t i = 0; i < get_num_server(); i++) {
      if (cache_servers.at(i)->check(req, &obj_type)) {
        n_copy++;
      }
    }
    return n_copy;
  }

  inline unsigned long get_cluster_id() {
    return this->cluster_params.cluster_id;
  };

  inline cacheServer *get_server(const unsigned long index) {
    return cache_servers.at(index);
  };

  void find_rep_factor(double *obj, double *byte, uint64_t *n_chunk_cnt,
                       uint64_t *n_server_obj, uint64_t *n_server_byte,
                       uint64_t *n_cluster_obj, uint64_t *n_cluster_byte);

  ~cacheCluster();
};
} // namespace CDNSimulator

#endif /* CACHE_CLUSTER_HPP */
