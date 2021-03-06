//
// Created by Juncheng Yang on 6/20/20.
//

#include "../include/libCacheSim/cache.h"
#include "../dataStructure/hashtable/hashtable.h"

static void *_get_func_handle(char *func_name, const char *const cache_name,
                                                            bool must_have, bool internal_func) {
    static void *handle = NULL;
    if (handle == NULL){
        handle = dlopen(NULL, RTLD_GLOBAL);
        /* should not check err here, otherwise ubuntu will report err even though
         * everything is OK
        char *err = dlerror();
        if (err != NULL){
            ERROR("error dlopen main program %s\n", err);
            abort();
        }
         */
    }

    char full_func_name[128];
    if (internal_func)
        sprintf(full_func_name, "_%s_%s", cache_name, func_name);
    else
        sprintf(full_func_name, "%s_%s", cache_name, func_name);

    void *func_ptr = dlsym(handle, full_func_name);

    if (must_have && func_ptr == NULL) {
        ERROR("unable to find %s error %s\n", full_func_name, dlerror());
        abort();
    }
    return func_ptr;
}

cache_t *cache_struct_init(const char *const cache_name,
                                                     common_cache_params_t params) {
    cache_t *cache = my_malloc(cache_t);
    memset(cache, 0, sizeof(cache_t));
    strncpy(cache->cache_name, cache_name, 32);
    cache->cache_size = params.cache_size;
    cache->cache_params = NULL;
    cache->default_ttl = params.default_ttl;
    int hash_power = HASH_POWER_DEFAULT;
    if (params.hash_power > 0 && params.hash_power < 40)
        hash_power = params.hash_power;
    cache->hashtable = create_hashtable(hash_power);
    hashtable_add_ptr_to_monitoring(cache->hashtable, &cache->list_head);
    hashtable_add_ptr_to_monitoring(cache->hashtable, &cache->list_tail);

    cache->cache_init =
            (cache_init_func_ptr)_get_func_handle("init", cache_name, true, false);
    cache->cache_free =
            (cache_free_func_ptr)_get_func_handle("free", cache_name, true, false);
    cache->get =
            (cache_get_func_ptr)_get_func_handle("get", cache_name, true, false);
    cache->check =
            (cache_check_func_ptr)_get_func_handle("check", cache_name, true, false);
    cache->insert =
            (cache_insert_func_ptr)_get_func_handle("insert", cache_name, true, false);
    cache->evict =
            (cache_evict_func_ptr)_get_func_handle("evict", cache_name, true, false);
    cache->remove_obj = (cache_remove_obj_func_ptr)_get_func_handle(
            "remove_obj", cache_name, false, false);

    return cache;
}

void cache_struct_free(cache_t *cache) {
    free_hashtable(cache->hashtable);
    my_free(sizeof(cache_t), cache);
}

cache_t *create_cache_with_new_size(cache_t *old_cache, gint64 new_size) {
    common_cache_params_t cc_params = {.cache_size = new_size,
                                                                         .default_ttl = old_cache->default_ttl};
    cache_t *cache = old_cache->cache_init(cc_params, old_cache->init_params);
    return cache;
}

cache_ck_res_e cache_check(cache_t *cache, request_t *req, bool update_cache,
                                                     cache_obj_t **cache_obj_ret) {
    cache_obj_t *cache_obj = hashtable_find(cache->hashtable, req);
//  if (cache_obj)
//    printf("check and find obj %lu size %lu\n", cache_obj->obj_id_int, cache_obj->obj_size);
    if (cache_obj_ret != NULL)
        *cache_obj_ret = cache_obj;
    if (cache_obj == NULL) {
        return cache_ck_miss;
    }

    cache_ck_res_e ret = cache_ck_hit;
#ifdef SUPPORT_TTL
    if (cache->default_ttl != 0) {
        if (cache_obj->exp_time < req->real_time) {
            ret = cache_ck_expired;
            if (likely(update_cache))
                cache_obj->exp_time =
                        req->real_time + (req->ttl != 0 ? req->ttl : cache->default_ttl);
        }
    }
#endif
    if (update_cache) {
        if (cache_obj->obj_size != req->obj_size) {
            cache->occupied_size -= cache_obj->obj_size;
            cache->occupied_size += req->obj_size;
            cache_obj->obj_size = req->obj_size;
        }
    }
    return ret;
}

cache_ck_res_e cache_get(cache_t *cache, request_t *req) {
#ifdef TRACK_EVICTION_AGE
    cache->n_req += 1;
#endif
//  VVVERBOSE("req %" PRIu64 ", obj %" PRIu64 ", obj_size %" PRIu64
//            ", cache size %" PRIu64 "/%" PRIu64 "\n",
//            cache->req_cnt, req->obj_id_int, req->obj_size,
//            cache->occupied_size, cache->cache_size);

//  static int cnt = 0;
//  cnt += 1;
//  if (cnt > 20)
//    abort();
//  DEBUG("host %llu get %llu size %lu\n", req->curr_host_id, req->obj_id_int, req->obj_size);

//  static FILE *f = NULL;
//  if (f == NULL)
//    f = fopen("cacheget", "w");
//  if (req->curr_host_id == 0)
//    fprintf(f, "%llu %llu\n", req->real_time, req->obj_id_int);


    cache_ck_res_e cache_check = cache->check(cache, req, true);
    if (req->obj_size <= cache->cache_size) {
        if (cache_check == cache_ck_miss) {
            cache->insert(cache, req);
        }
        while (cache->occupied_size > cache->cache_size)
            cache->evict(cache, req, NULL);
    } else {
        WARNING("req %lld: obj size %ld larger than cache size %ld\n",
                        (long long)cache->req_cnt, (long)req->obj_size,
                        (long)cache->cache_size);
    }

//  assert(cache->check(cache, req, true) == cache_ck_hit);
    cache->req_cnt += 1;
    return cache_check;
}

cache_obj_t *cache_insert_LRU(cache_t *cache, request_t *req) {
#ifdef SUPPORT_TTL
    if (cache->default_ttl != 0 && req->ttl == 0) {
        req->ttl = cache->default_ttl;
    }
#endif
    cache->occupied_size += req->obj_size;
    cache_obj_t *cache_obj = hashtable_insert(cache->hashtable, req);
#ifdef TRACK_EVICTION_AGE
    cache_obj->create_time = cache->n_req;
#endif

    cache->n_obj += 1;
    if (unlikely(cache->list_head == NULL)) {
        // an empty list, this is the first insert
        cache->list_head = cache_obj;
        cache->list_tail = cache_obj;
    } else {
        cache->list_tail->list_next = cache_obj;
        cache_obj->list_prev = cache->list_tail;
    }
    cache->list_tail = cache_obj;
    return cache_obj;
}

void cache_evict(cache_t *cache, request_t *req, cache_obj_t *evicted_obj) {
    cache_obj_t *obj_to_evict = cache->list_head;
//  printf("evict %lu, current size %lu n_obj %lu\n",
//      obj_to_evict->obj_id_int, cache->occupied_size, cache->n_obj);
#ifdef TRACK_EVICTION_AGE
    cache->eviction_age_sum += cache->n_req - obj_to_evict->create_time;
#endif
//  static FILE *f = NULL;
//  if (f == NULL)
//    f = fopen("evict", "w");
//  fprintf(f, "%llu\n", obj_to_evict->obj_id_int);

    cache->n_obj -= 1;
    DEBUG_ASSERT(cache->n_obj >= 0);
    if (evicted_obj != NULL) {
        // return evicted object to caller
        memcpy(evicted_obj, obj_to_evict, sizeof(cache_obj_t));
    }
    DEBUG_ASSERT(cache->list_head != cache->list_head->list_next);
    cache->list_head = cache->list_head->list_next;
    cache->list_head->list_prev = NULL;
    DEBUG_ASSERT(cache->occupied_size >= obj_to_evict->obj_size);
    cache->occupied_size -= obj_to_evict->obj_size;
    hashtable_delete(cache->hashtable, obj_to_evict);
    DEBUG_ASSERT(cache->list_head != cache->list_head->list_next);
    /** obj_to_evict is not freed or returned to hashtable, if you have
     * extra_metadata allocated with obj_to_evict, you need to free them now,
     * otherwise, there will be memory leakage **/
}

cache_obj_t *cache_get_obj(cache_t *cache, request_t *req) {
    return hashtable_find(cache->hashtable, req); 
}

