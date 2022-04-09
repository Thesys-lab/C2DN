//
// Created by Juncheng Yang on 2019-06-21.
//

#ifndef CONSISTENT_HASH_H_
#define CONSISTENT_HASH_H_

#ifdef __cplusplus /* If this is a C++ compiler, use C linkage */
extern "C" {
#endif


#include <stdlib.h>
#include <stdio.h>
#include <string.h>

#include <math.h>


#define N_VNODE_PER_SERVER 160

typedef struct {
    unsigned int point;  // point on ring
    unsigned int server_id;
} vnode_t;

typedef struct {
    unsigned int n_point;
    unsigned int n_server;
    vnode_t *vnodes;
} ring_t;

//typedef int (*compfn)(const void *, const void *);

ring_t *ch_ring_create(const int n_server,
                       const uint32_t *weight);

vnode_t *ch_ring_get_server(const uint64_t *const key,
                            const ring_t *const ring);

void ch_ring_get_servers(const uint64_t *const key,
                         const ring_t *const ring,
                         const uint32_t n,
                         uint32_t *ret_idxs,
                         uint32_t *original_lead_server,
                         uint32_t *capacity);

void ch_ring_destroy(ring_t *ring);

#ifdef __cplusplus /* If this is a C++ compiler, end C linkage */
}
#endif

#endif // CONSISTENT_HASH_H_
