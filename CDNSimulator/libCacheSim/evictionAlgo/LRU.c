//
//  a LRU module that supports different obj size
//
//
//  LRU.c
//  libCacheSim
//
//  Created by Juncheng on 12/4/18.
//  Copyright © 2018 Juncheng. All rights reserved.
//

#ifdef __cplusplus
extern "C" {
#endif

#include "../include/libCacheSim/evictionAlgo/LRU.h"
#include "include/cacheAlg.h"

cache_t *LRU_init(common_cache_params_t ccache_params, void *init_params) {
  cache_t *cache = cache_struct_init("LRU", ccache_params);
  return cache;
}

void LRU_free(cache_t *cache) {
  cache_struct_free(cache);
}

cache_ck_res_e LRU_check(cache_t *cache, request_t *req, bool update_cache) {
  cache_obj_t *cache_obj;
  cache_ck_res_e ret = cache_check(cache, req, update_cache, &cache_obj);

  if (cache_obj && likely(update_cache)) {
    /* lru_tail is the newest, move cur obj to lru_tail */
    move_obj_to_tail(&cache->list_head, &cache->list_tail, cache_obj);
  }
  return ret;
}

cache_ck_res_e LRU_get(cache_t *cache, request_t *req) {
  return cache_get(cache, req);
}

void LRU_insert(cache_t *LRU, request_t *req) {
  cache_insert_LRU(LRU, req);
}

void LRU_evict(cache_t *cache, request_t *req, cache_obj_t *evicted_obj) {
  cache_evict(cache, req, evicted_obj);
}

void LRU_remove_obj(cache_t *cache, cache_obj_t *obj_to_remove) {
  cache_obj_t *cache_obj = hashtable_find_obj(cache->hashtable, obj_to_remove);
  if (cache_obj == NULL) {
    WARNING("obj to remove is not in the cache\n");
    return;
  }
  remove_obj_from_list(&cache->list_head, &cache->list_tail, cache_obj);
  hashtable_delete(cache->hashtable, cache_obj);

  assert(cache->occupied_size >= cache_obj->obj_size);
  cache->occupied_size -= cache_obj->obj_size;
}

#ifdef __cplusplus
extern "C" {
#endif
