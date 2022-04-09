//
//  cacheServer.cpp
//  CDNSimulator
//
//  Created by Juncheng on 7/11/17.
//  Copyright © 2017 Juncheng. All rights reserved.
//

#ifndef CACHE_SERVER_HPP
#define CACHE_SERVER_HPP

#ifdef __cplusplus
extern "C" {
#endif

#include <libCacheSim/logging.h>
#include <libCacheSim/struct.h>
#include <libCacheSim/cache.h>

#include <glib.h>
#include <stdlib.h>

#ifdef __cplusplus
}
#endif

#include <atomic>
#include <fstream>
#include <iostream>
#include <mutex>
#include <ostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <unordered_map>

#include "constCDNSimulator.hpp"
#include "stat.hpp"
#include "C2DNconfig.h"

namespace CDNSimulator {

typedef struct {
  unsigned int obj_type: 4;
  unsigned int chunk_id: 14;
} cached_obj_data_t;

typedef enum {
  main_cache = 1,
  gutter_cache,
  unknown_cache,

  invalid_cache,
} cache_type_e;

typedef struct {
  uint32_t server_id;
  std::string server_name;
  uint64_t cache_size;
  std::string cache_alg;
  void *cache_alg_params;

  /* related to erasure coding */
  unsigned int EC_n;
  unsigned int EC_k;

  /* fraction of size for gutter cache */
  double gutter_space;

} server_params_t;

class cacheServer {

 public:
  server_params_t server_params;
  int gutter_prob = 100;

  cache_t *cache;
  cache_t *gcache;

  /* availability simulation */
  bool _available = true;

  uint64_t req_cnt = 0;
  uint64_t hit_cnt = 0;
  uint64_t gutter_hit_cnt = 0;
  uint64_t gutter_req_cnt = 0;

#ifdef TRACK_BYTE_REUSE
  std::unordered_map<uint64_t, uint64_t> obj_byte;
  std::unordered_map<uint64_t, uint64_t> obj_reuse_byte;
#endif

  cacheServer(server_params_t params);

  inline bool get(request_t *req, cache_type_e cache_type) {
    if (!_available)
      return false;

    cache_ck_res_e ck;
    req->curr_host_id = server_params.server_id;

    if (cache_type == gutter_cache) {
      ck = this->gcache->get(this->gcache, (request_t *) req);
    } else if (cache_type == main_cache) {
      ck = this->cache->get(this->cache, (request_t *) req);
    } else if (cache_type == unknown_cache) {
      if (this->gcache != nullptr && cache_get_obj(this->gcache, (request_t *) req)) {
        ck = this->gcache->get(this->gcache, (request_t *) req);
      } else {
        ck = this->cache->get(this->cache, (request_t *) req);
      }
    } else {
      printf("unknown_obj_type cache type %d\n", cache_type);
      abort();
    }

    assert(check(req, nullptr));
    return ck == cache_ck_hit;
  }

  inline bool check(const request_t *req, obj_type_e *obj_type) {
    if (!_available)
      return false;

    req_cnt += 1;
    cache_obj_t *cache_obj = cache_get_obj(this->cache, (request_t *) req);
    if (cache_obj)
      hit_cnt += 1;

    if (cache_obj == nullptr && this->gcache != nullptr) {
      cache_obj = cache_get_obj(this->gcache, (request_t *) req);
      gutter_req_cnt += 1;
      if (cache_obj)
        gutter_hit_cnt += 1;
    }

    if (cache_obj && obj_type) {
        *obj_type = static_cast<obj_type_e>(cache_obj->extra_metadata_u8[0]);
    }

    return cache_obj != nullptr;
  }

  inline obj_type_e get_obj_type(const request_t *const req) {
    cache_obj_t *cache_obj = cache_get_obj(this->cache, (request_t *) req);
    if (cache_obj == nullptr) {
      assert(this->gcache != nullptr);
      cache_obj = cache_get_obj(this->gcache, (request_t *) req);
    }
    assert(cache_obj != nullptr);

    return static_cast<obj_type_e>(cache_obj->extra_metadata_u8[0]);
  }

  inline void set_obj_type(const request_t *const req, obj_type_e obj_type) {
    cache_obj_t *cache_obj = cache_get_obj(this->cache, (request_t *) req);
    if (cache_obj == nullptr) {
      assert(this->gcache != nullptr);
      cache_obj = cache_get_obj(this->gcache, (request_t *) req);
    }
    assert(cache_obj != nullptr);

    cache_obj->extra_metadata_u8[0] = static_cast<uint8_t>(obj_type);
  }

  inline int get_chunk_id(const request_t *const req) {
    cache_obj_t *cache_obj = cache_get_obj(this->cache, (request_t *) req);
    if (cache_obj == nullptr) {
      assert(this->gcache != nullptr);
      cache_obj = cache_get_obj(this->gcache, (request_t *) req);
    }
    assert(cache_obj != nullptr);

    return static_cast<int>(cache_obj->extra_metadata_u8[1]);
  }

  inline void set_chunk_id(const request_t *const req, int chunk_id) {
    cache_obj_t *cache_obj = cache_get_obj(this->cache, (request_t *) req);
    if (cache_obj == nullptr) {
      assert(this->gcache != nullptr);
      cache_obj = cache_get_obj(this->gcache, (request_t *) req);
    }
    assert(cache_obj != nullptr);

    cache_obj->extra_metadata_u8[0] = static_cast<int8_t>(chunk_obj);
    cache_obj->extra_metadata_u8[1] = static_cast<int8_t>(chunk_id);
  }

  inline bool set_unavailable() {
    _available = false;
    return _available;
  }

  inline bool set_available() {
    _available = true;
    return _available;
  }

  inline bool is_available() { return _available; }

  inline unsigned long get_server_id() { return this->server_params.server_id; }

  void print_cache() {
#ifdef TRACE_EVICTION_AGE
    cout << fixed << setprecision(1) << "server " << server_id << ",\t"
       << cache->n_req << " req,\t eviction age " << (double) cache->eviction_age_sum / cache->n_req
       << ",\t eviction age/cache size " << (double) cache->eviction_age_sum / cache->n_req / cache->cache_size
       << ",\t used size (GB) " << (double) cache->occupied_size / GB << "/" << (double) cache->cache_size / GB << endl;
#else
    std::cout << std::fixed << std::setprecision(1) << "server " << server_params.server_id
         << ",\t used size (GB) " << (double) cache->occupied_size / GB << "/" << (double) cache->cache_size / GB;

    if (gcache != nullptr)
      std::cout << ",\t" << (double) gcache->occupied_size / GB << "/"
           << (double) gcache->cache_size / GB;

    std::cout << std::setprecision(4);
    std::cout << ", \thitCnt/reqCnt " << hit_cnt << "/" << req_cnt << "(" << (double) hit_cnt/req_cnt << ")";

    if (gcache != nullptr)
      std::cout << ",\tgutterHitCnt/gutterReqCnt " << gutter_hit_cnt << "/" << gutter_req_cnt
          << "(" << (double) gutter_hit_cnt/gutter_req_cnt << ")";

    std::cout << std::endl;
#endif
  }

  ~cacheServer() {
    print_cache();
    cache->cache_free(cache);
  }
};

} // namespace CDNSimulator

#endif /* CACHE_SERVER_HPP */
