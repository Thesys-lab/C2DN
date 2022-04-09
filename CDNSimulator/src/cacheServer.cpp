//
//  cacheServer.cpp
//  CDNSimulator
//
//  Created by Juncheng on 11/18/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#include <iomanip>
#include "cacheServer.hpp"
#include <iostream>
#include "libCacheSim/evictionAlgo.h"
#include "libCacheSim/cacheObj.h"
#include "libCacheSim/struct.h"
#include "C2DNconfig.h"

using namespace std;

namespace CDNSimulator {

/************************** cacheServer *****************************
 this class is the cache server class, it simulates a cache server
 ********************************************************************/

/**
 *
 * @param params
 */
cacheServer::cacheServer(server_params_t params) {

  this->server_params = params;
  common_cache_params_t ccache_params;
  ccache_params.cache_size = (long long) params.cache_size;
  ccache_params.default_ttl = 0;

#ifdef __APPLE__
  /* for faster debug on my mac */
  ccache_params.hash_power = 18;
#else
  ccache_params.hash_power = 24;
#endif

  this->gcache = nullptr;
  if (params.gutter_space > 0.00001) {
    ccache_params.cache_size = (long long) (params.gutter_space * params.cache_size);
    this->gcache = LRU_init(ccache_params, NULL);
  }

  ccache_params.cache_size = (long long) ((1 - params.gutter_space) * params.cache_size);


  if (params.cache_alg == "lru") {
    this->cache = LRU_init(ccache_params, NULL);
  } else if (params.cache_alg == "fifo") {
    this->cache = FIFO_init(ccache_params, NULL);
  } else {
    ERROR("unknown cache replacement algorithm %s\n", params.cache_alg.c_str());
    abort();
  }

  srand((unsigned int) time(nullptr));
}


} // namespace CDNSimulator
