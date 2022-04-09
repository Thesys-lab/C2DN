//
// Created by Juncheng Yang on 2019-06-21.
//
// modified from libketama
//

#include <vector>
#include <assert.h>

#include "../libCacheSim/dataStructure/hash/hash.h"

#include "consistentHash.hpp"
#include "md5.h"
#include "C2DNconfig.h"


int ch_ring_compare(const void *a, const void *b) {
    vnode_t *node_a = (vnode_t *) a;
    vnode_t *node_b = (vnode_t *) b;
    return (node_a->point < node_b->point) ? -1 : ((node_a->point > node_b->point) ? 1 : 0);
}

void md5_digest(const char *const inString, unsigned char md5pword[16]) {
    md5_state_t md5state;

    md5_init(&md5state);
    md5_append(&md5state, (unsigned char *) inString, (int) strlen(inString));
    md5_finish(&md5state, md5pword);
}

unsigned int ketama_hash(const char *const inString) {
    unsigned char digest[16];
    md5_digest(inString, digest);
    return (unsigned int) ((digest[3] << 24) | (digest[2] << 16) | (digest[1] << 8) | digest[0]);
}

ring_t *ch_ring_create(const int n_server, const uint32_t *weight) {
    vnode_t *vnodes = (vnode_t *) malloc(sizeof(vnode_t) * n_server * N_VNODE_PER_SERVER);

    double total_weight = 0;
    for (int i = 0; i < n_server; i++)
        total_weight += weight[i];

    ring_t *ring = (ring_t *) malloc(sizeof(ring_t));
    ring->n_server = n_server;
    ring->n_point = n_server * N_VNODE_PER_SERVER;
    ring->vnodes = vnodes;

    int i;
    unsigned int k, cnt = 0;

    for (i = 0; i < n_server; i++) {
        // default all servers have the same weight
        unsigned int ks = N_VNODE_PER_SERVER / 4;
        if (weight != nullptr) {
            ks = (unsigned int) floorf((double) weight[i] / total_weight * n_server * (N_VNODE_PER_SERVER / 4));
        }

        for (k = 0; k < ks; k++) {
            /* 40 hashes, 4 numbers per hash = 160 points per server */
            char ss[30];
            unsigned char digest[16];

            sprintf(ss, "%u-%u", i, k);
            md5_digest(ss, digest);

            /* Use successive 4-bytes from hash as numbers for the points on the circle: */
            int h;
            for (h = 0; h < 4; h++) {
                vnodes[cnt].point = (digest[3 + h * 4] << 24) | (digest[2 + h * 4] << 16)
                                    | (digest[1 + h * 4] << 8) | digest[h * 4];

                vnodes[cnt].server_id = i;
                cnt++;
            }
        }
    }

    /* Sorts in ascending order of "point" */
    qsort((void *) vnodes, cnt, sizeof(vnode_t), ch_ring_compare);

    return ring;
}

int ch_ring_get_vnode_idx(const uint64_t *const key, const ring_t *const ring) {
    uint64_t h = get_hash_value_int_64(key) & 0x00000000ffffffff;
    vnode_t *vnodes = ring->vnodes;
    int lowp = 0;
    int highp = ring->n_point;
    uint64_t midp;
    uint64_t midval, midval_prev;
    int vnode_idx = -1;

    // divide and conquer array search to find server with next biggest
    // point after what this key hashes to
    while (true) {
        midp = (int) ((lowp + highp) / 2);

        if (midp == ring->n_point) {
            vnode_idx = 0;
            break;
        }

        midval = vnodes[midp].point;
        midval_prev = midp == 0 ? 0 : vnodes[midp - 1].point;

        if (h <= midval && h > midval_prev) {
            vnode_idx = midp;
            break;
        } else {
            if (h > midval) {
                lowp = midp + 1;
            } else {
                highp = midp - 1;
            }

            if (lowp > highp) {
                vnode_idx = 0;
                break;
            }
        }

        if (vnode_idx != -1)
            break;
    }

    return vnode_idx;
}

vnode_t *ch_ring_get_server(const uint64_t *const key, const ring_t *const ring) {
    return ring->vnodes + ch_ring_get_vnode_idx(key, ring);
}


void ch_ring_get_servers(const uint64_t *const key,
                         const ring_t *const ring,
                         const uint32_t n,
                         uint32_t *ret_idxs,
                         uint32_t *original_lead_server,
                         uint32_t *capacity) {

    vnode_t *vnodes = ring->vnodes;
    int start_vnode_idx = ch_ring_get_vnode_idx(key, ring);
    if (original_lead_server != NULL)
        *original_lead_server = (uint32_t) (vnodes[(start_vnode_idx) % (ring->n_point)].server_id);

    unsigned int i = 0, vnode_pos = 0;
    unsigned int server_id;
    uint32_t checked_server[ring->n_server];
    memset(checked_server, 0, sizeof(uint32_t) * ring->n_server);
    unsigned int picked_n_unavailable = 0;

    while (i < n) {
        server_id = vnodes[(start_vnode_idx + vnode_pos) % (ring->n_point)].server_id;
        if (checked_server[server_id] == 0) {
            if (capacity == NULL) {
                ret_idxs[i] = server_id;
                i++;
            } else {
                if (capacity[server_id] > 0) {
                    ret_idxs[i] = server_id;
                    i++;
                    capacity[server_id] -= 1;
                } else {
                    picked_n_unavailable += 1;
                }
            }
        }

        checked_server[server_id] = 1;
        vnode_pos++;

        if (vnode_pos > ring->n_point) {
            printf("ERROR: searched all %u points on the consistent hash ring, but cannot find %u available servers\n",
                   ring->n_point, n);

            printf("server_capacity: ");
            for (unsigned int j = 0; j < ring->n_server; j++)
                printf("%d ", capacity[j]);
            printf("\n");

            printf("checked server: ");
            for (unsigned int j = 0; j < ring->n_server; j++)
                printf("%d ", checked_server[j]);
            printf("\n");

            printf("picked servers: ");
            for (unsigned j = 0; j < n; j++)
                printf("%d, ", ret_idxs[j]);
            printf("\n");
            abort();
        }
    }
}


void ch_ring_destroy(ring_t *ring) {
    free(ring->vnodes);
    free(ring);
}
