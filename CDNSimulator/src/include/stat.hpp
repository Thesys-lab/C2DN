//
//  stat.hpp
//  CDNSimulator
//
//  Created by Juncheng on 11/15/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#ifndef STAT_HPP
#define STAT_HPP

#include <glib.h>
#include <stdio.h>

#include "constCDNSimulator.hpp"
#include <atomic>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>
#include <iomanip>

namespace CDNSimulator {
class cacheServerStat {
 public:
  unsigned long server_id;

  unsigned long req_cnt = 0;
  unsigned long hit_cnt = 0;
  unsigned long gutter_hit_cnt = 0;
  unsigned long gutter_req_cnt = 0;

  unsigned long long req_bytes = 0;
  unsigned long long hit_bytes = 0;
  double omr = 0;
  double bmr = 0;
  bool available = true;

  unsigned long cache_size;
  unsigned long used_cache_size = 0;

  cacheServerStat(const unsigned long server_id, const unsigned long cache_size)
      : server_id(server_id), req_cnt(0), hit_cnt(0), cache_size(cache_size) {};

  static std::string stat_str_header() {
    return "server_id, hitCnt/reqCnt, gutter hitCnt/reqCnt, \n";
  }

  std::string stat_str(bool header = true) {
    std::stringstream ss;
    ss.precision(4);
    if (header)
      ss << stat_str_header();
    ss << server_id << ", " <<
        hit_cnt << "/" << req_cnt << ", " << gutter_hit_cnt << ", " << gutter_req_cnt
       << std::endl;
    return ss.str();
  }

  void cal_stat() {
    omr = (double) hit_cnt / req_cnt;
    bmr = (double) hit_bytes / req_bytes;
  }

  ~cacheServerStat() = default;

  cacheServerStat &operator=(const cacheServerStat &stat) {
    if (this == &stat)
      return *this;
    server_id = stat.server_id;
    cache_size = stat.cache_size;
    req_cnt = stat.req_cnt;
    hit_cnt = stat.hit_cnt;

    return *this;
  }
};

class cacheClusterStat {
 public:
  unsigned long cluster_id;
  unsigned long n_server;
  unsigned long n_avail_server;
  unsigned long *server_cache_sizes;
  unsigned long cluster_cache_size = 0;
  unsigned long cluster_cache_size_in_gb = 0;
  unsigned long cluster_cache_size_in_mb = 0;

  /** the number of requests to each sever in this cluster, if weight is equal,
   *  then the number of requests routed to each server should be similar,
   *  but due to the skewness of request pattern, it may not be very close */

  /** the is_hit count of this cluster */
  unsigned long cluster_hit_cnt = 0;
  unsigned long cluster_req_cnt = 0;
  unsigned long long cluster_hit_bytes = 0;
  unsigned long long cluster_req_bytes = 0;

  unsigned long full_obj_hit_cnt = 0;
  unsigned long long full_obj_hit_bytes = 0;
  unsigned long chunk_obj_hit_cnt = 0;
  unsigned long long chunk_obj_hit_bytes = 0;

//  double rep_coef_obj = 0;
//  double rep_coef_byte = 0;

  unsigned long *n_miss;

  unsigned int EC_n;
  unsigned int EC_k;

  unsigned long long midgress_bytes = 0;
  unsigned long long intra_cluster_bytes = 0;

#ifdef TRACK_EXCLUSIVE_HIT
  unsigned long long exclusive_hit_cnt  = 0;
  unsigned long long exclusive_hit_byte = 0;
#endif

  cacheClusterStat(const unsigned long cluster_id, const unsigned long n_server,
                   const unsigned long *cache_sizes, const unsigned int EC_n,
                   const unsigned int EC_k)
      : cluster_id(cluster_id), n_server(n_server), n_avail_server(n_server),
        EC_n(EC_n), EC_k(EC_k) {

    // add () to new to initialize the array to 0,
    // https://stackoverflow.com/questions/7546620/operator-new-initializes-memory-to-zero
    server_cache_sizes = new unsigned long[n_server]();
    n_miss = new unsigned long[EC_n]();

    for (unsigned int i = 0; i < n_server; i++) {
      server_cache_sizes[i] = cache_sizes[i];
      cluster_cache_size += cache_sizes[i];
    }

    // add 10 to round up in heterogeneous casse
    cluster_cache_size_in_gb = (cluster_cache_size + 10) / (1000 * 1000 * 1000);
    cluster_cache_size_in_mb = (cluster_cache_size + 10) / (1000 * 1000);
  }

  cacheClusterStat(const cacheClusterStat &stat) = delete;
  cacheClusterStat &operator=(const cacheClusterStat &stat) = delete;

  ~cacheClusterStat() {
    delete[] server_cache_sizes;
    delete[] n_miss;
  }

  static std::string stat_str_header(bool with_server = false) {
    if (with_server)
      return "cluster_id, #available servers/#servers, "
             "hit_cnt/req_cnt, midgress/req_bytes, per server "
             "(hit_cnt/req_cnt)\n";
    else
      return "cluster_id, #available servers/#servers, "
             "hit_cnt/req_cnt, midgress/req_bytes\n";
  }

  std::string stat_str(bool header = false, bool with_server = false) {
    std::stringstream ss;
    ss.precision(4);
    if (header)
      ss << stat_str_header();

    ss << cluster_id << ", \t" << n_avail_server << "/" << n_server << ", \t"
       << cluster_hit_cnt << "/" << cluster_req_cnt << ", \t"
       << midgress_bytes << "/" << cluster_req_bytes;
    ss << "\n";

    return ss.str();
  }

  std::string final_stat_header() {
    return "n/k, cluster_cache_size, miss ratio (omr/bmr), "
           "full_obj_hit_cnt/chunk_obj_hit_cnt/cluster_hit_cnt/cluster_req_cnt, "
           "full_obj_hit_bytes/chunk_obj_hit_bytes/cluster_hit_bytes/"
           "cluster_req_bytes, "
           "midgress/intra_cluster, n_chunk_miss\n";
  }

  std::string final_stat() {
    double omr = 1 - (double) cluster_hit_cnt / cluster_req_cnt;
    double bmr = 1 - (double) cluster_hit_bytes / cluster_req_bytes;
    std::stringstream ss;

    ss << std::fixed << std::setprecision(4) << EC_n << "/" << EC_k
       << ", " << (double) cluster_cache_size / 1000000000 << "GB"
       << ", " << omr << "/" << bmr << ", "
       << full_obj_hit_cnt << "/" << chunk_obj_hit_cnt << "/" << cluster_hit_cnt
       << "/" << cluster_req_cnt << ", " << full_obj_hit_bytes << "/"
       << chunk_obj_hit_bytes << "/" << cluster_hit_bytes << "/"
       << cluster_req_bytes << ", " << midgress_bytes << "/"
       << intra_cluster_bytes << ", ";

    ss << n_miss[0];
    for (unsigned i = 1; i < EC_n; i++)
      ss << "/" << n_miss[i];
    ss << "\n";

#ifdef TRACK_EXCLUSIVE_HIT
    ss << "exclusive_hit_cnt, exclusive_hit_byte\n" << exclusive_hit_cnt << ", " << exclusive_hit_byte << "\n";
#endif
    return ss.str();
  }
};

struct bucketStat {
  uint64_t bucket_id;
  uint64_t req_cnt = 0;
  uint64_t hit_cnt = 0;
  uint64_t req_byte = 0;
  uint64_t midgress = 0;

  uint64_t protected_req_cnt = 0;
  uint64_t protected_hit_cnt = 0;
  uint64_t unprotected_req_cnt = 0;
  uint64_t unprotected_hit_cnt = 0;

  uint64_t protected_req_byte = 0;
  uint64_t protected_miss_byte = 0;
  uint64_t unprotected_req_byte = 0;
  uint64_t unprotected_miss_byte = 0;

  static std::string stat_str_header() {
    return "#bucket_id, hit_cnt/req_cnt, midgress/req_bytes\n";
  }

  std::string stat_str() {
    std::stringstream ss;
    ss.precision(4);
    ss << hit_cnt << "/" << req_cnt << "/" << midgress << "/" << req_byte;
    return ss.str();
  }

  std::string stat_str_unprotected() {
    std::stringstream ss;
    ss.precision(4);
    ss << unprotected_hit_cnt << "/" << unprotected_req_cnt << "/" << unprotected_miss_byte << "/"
       << unprotected_req_byte;
    return ss.str();
  }
  std::string stat_str_protected() {
    std::stringstream ss;
    ss.precision(4);
    ss << protected_hit_cnt << "/" << protected_req_cnt << "/" << protected_miss_byte << "/" << protected_req_byte;
    return ss.str();
  }
  void clear() {
    req_cnt = 0;
    hit_cnt = 0;
    req_byte = 0;
    midgress = 0;

    protected_req_cnt = 0;
    protected_hit_cnt = 0;
    unprotected_req_cnt = 0;
    unprotected_hit_cnt = 0;

    protected_req_byte = 0;
    protected_miss_byte = 0;
    unprotected_req_byte = 0;
    unprotected_miss_byte = 0;
  }
};

} // namespace CDNSimulator

#endif /* STAT_HPP */
