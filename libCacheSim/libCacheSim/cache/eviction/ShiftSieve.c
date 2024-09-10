

#include "../../dataStructure/hashtable/hashtable.h"
#include "../../include/libCacheSim/cache.h"

#ifdef __cplusplus
extern "C" {
#endif

static const uint32_t VISITED_MASK  = 1U << 0;  // 01
static const uint32_t SURVIVED_MASK = 1U << 1;  // 10

typedef struct {
  cache_obj_t *q_head;
  cache_obj_t *q_tail;

  cache_obj_t *pointer;
  int64_t     right;
  int64_t     hot;
} ShiftSieve_params_t;

// ***********************************************************************
// ****                                                               ****
// ****                   function declarations                       ****
// ****                                                               ****
// ***********************************************************************
static void ShiftSieve_free(cache_t *cache);
static bool ShiftSieve_get(cache_t *cache, const request_t *req);
static cache_obj_t *ShiftSieve_find(cache_t *cache, const request_t *req,
                               const bool update_cache);
static cache_obj_t *ShiftSieve_insert(cache_t *cache, const request_t *req);
static cache_obj_t *ShiftSieve_to_evict(cache_t *cache, const request_t *req);
static void ShiftSieve_evict(cache_t *cache, const request_t *req);
static bool ShiftSieve_remove(cache_t *cache, const obj_id_t obj_id);

// ***********************************************************************
// ****                                                               ****
// ****                   end user facing functions                   ****
// ****                                                               ****
// ****                       init, free, get                         ****
// ***********************************************************************

/**
 * @brief initialize cache
 *
 * @param ccache_params some common cache parameters
 * @param cache_specific_params cache specific parameters, see parse_params
 * function or use -e "print" with the cachesim binary
 */
cache_t *ShiftSieve_init(const common_cache_params_t ccache_params,
                    const char *cache_specific_params) {
  cache_t *cache =
      cache_struct_init("ShiftSieve", ccache_params, cache_specific_params);
  cache->cache_init = ShiftSieve_init;
  cache->cache_free = ShiftSieve_free;
  cache->get = ShiftSieve_get;
  cache->find = ShiftSieve_find;
  cache->insert = ShiftSieve_insert;
  cache->evict = ShiftSieve_evict;
  cache->remove = ShiftSieve_remove;
  cache->to_evict = ShiftSieve_to_evict;

  if (ccache_params.consider_obj_metadata) {
    cache->obj_md_size = 1;
  } else {
    cache->obj_md_size = 0;
  }

  cache->eviction_params = my_malloc(ShiftSieve_params_t);
  memset(cache->eviction_params, 0, sizeof(ShiftSieve_params_t));
  ShiftSieve_params_t *params = (ShiftSieve_params_t *)cache->eviction_params;
  params->q_head = NULL;
  params->q_tail = NULL;
  params->pointer = NULL;
  params->right = 0;
  params->hot = 0;

  return cache;
}

/**
 * free resources used by this cache
 *
 * @param cache
 */
static void ShiftSieve_free(cache_t *cache) {
  free(cache->eviction_params);
  cache_struct_free(cache);
}

/**
 * @brief this function is the user facing API
 * it performs the following logic
 *
 * ```
 * if obj in cache:
 *    update_metadata
 *    return true
 * else:
 *    if cache does not have enough space:
 *        evict until it has space to insert
 *    insert the object
 *    return false
 * ```
 *
 * @param cache
 * @param req
 * @return true if cache hit, false if cache miss
 */

static bool ShiftSieve_get(cache_t *cache, const request_t *req) {
  bool ck_hit = cache_get_base(cache, req);
  return ck_hit;
}

// ***********************************************************************
// ****                                                               ****
// ****       developer facing APIs (used by cache developer)         ****
// ****                                                               ****
// ***********************************************************************

/**
 * @brief find an object in the cache
 *
 * @param cache
 * @param req
 * @param update_cache whether to update the cache,
 *  if true, the object is promoted
 *  and if the object is expired, it is removed from the cache
 * @return the object or NULL if not found
 */
static cache_obj_t *ShiftSieve_find(cache_t *cache, const request_t *req,
                               const bool update_cache) {
  ShiftSieve_params_t *params = (ShiftSieve_params_t *)cache->eviction_params;
  cache_obj_t *cache_obj = cache_find_base(cache, req, update_cache);
  if (cache_obj != NULL && update_cache) {
    if ((cache_obj->shiftsieve.status & SURVIVED_MASK) == 0) {
      if (cache_obj == params->pointer) {
        params->pointer = cache_obj->queue.prev;
      }
      move_obj_to_head(&params->q_head, &params->q_tail, cache_obj);
    }
    cache_obj->shiftsieve.status |= VISITED_MASK;
  }

  return cache_obj;
}

/**
 * @brief insert an object into the cache,
 * update the hash table and cache metadata
 * this function assumes the cache has enough space
 * eviction should be
 * performed before calling this function
 *
 * @param cache
 * @param req
 * @return the inserted object
 */
static cache_obj_t *ShiftSieve_insert(cache_t *cache, const request_t *req) {
  ShiftSieve_params_t *params = cache->eviction_params;
  cache_obj_t *obj = cache_insert_base(cache, req);
  prepend_obj_to_head(&params->q_head, &params->q_tail, obj);
  obj->shiftsieve.status = 0;

  return obj;
}

static cache_obj_t *ShiftSieve_to_evict(cache_t *cache, const request_t *req) {
  ShiftSieve_params_t *params = cache->eviction_params;

  /* if we have run one full around or first eviction */
  cache_obj_t *obj = params->pointer;
  if (obj == NULL) {
    obj = params->q_tail;
    params->right = 0;
    params->hot = 0;
  }

  while (obj->shiftsieve.status & VISITED_MASK) {
    obj->shiftsieve.status &= ~VISITED_MASK;
    if ((obj->shiftsieve.status & SURVIVED_MASK) == 0) {
      obj->shiftsieve.status |= SURVIVED_MASK;
      params->hot++;
    }
    obj = obj->queue.prev;
    params->right++;
    if (cache->n_obj - params->right <= params->hot / 2) {
      obj = params->q_tail;
      params->right = 0;
      params->hot = 0;
    }
  }
  params->pointer = obj->queue.prev;
  return obj;
}

/**
 * @brief evict an object from the cache
 * it needs to call cache_evict_base before returning
 * which updates some metadata such as n_obj, occupied size, and hash table
 *
 * @param cache
 * @param req not used
 * @param evicted_obj if not NULL, return the evicted object to caller
 */
static void ShiftSieve_evict(cache_t *cache, const request_t *req) {
  ShiftSieve_params_t *params = cache->eviction_params;

  /* if we have run one full around or first eviction */
  cache_obj_t *obj = params->pointer;
  if (obj == NULL) {
    obj = params->q_tail;
    params->right = 0;
    params->hot = 0;
  }

  while (obj->shiftsieve.status & VISITED_MASK) {
    obj->shiftsieve.status &= ~VISITED_MASK;
    if ((obj->shiftsieve.status & SURVIVED_MASK) == 0) {
      obj->shiftsieve.status |= SURVIVED_MASK;
      params->hot++;
    }
    obj = obj->queue.prev;
    params->right++;
    if (cache->n_obj - params->right <= params->hot / 2) {
      obj = params->q_tail;
      params->right = 0;
      params->hot = 0;
    }
  }

  params->pointer = obj->queue.prev;
  remove_obj_from_list(&params->q_head, &params->q_tail, obj);
  cache_evict_base(cache, obj, true);
}

static void ShiftSieve_remove_obj(cache_t *cache, cache_obj_t *obj_to_remove) {
  DEBUG_ASSERT(obj_to_remove != NULL);
  ShiftSieve_params_t *params = cache->eviction_params;
  if (obj_to_remove == params->pointer) {
    params->pointer = obj_to_remove->queue.prev;
  }
  remove_obj_from_list(&params->q_head, &params->q_tail, obj_to_remove);
  cache_remove_obj_base(cache, obj_to_remove, true);
}

/**
 * @brief remove an object from the cache
 * this is different from cache_evict because it is used to for user trigger
 * remove, and eviction is used by the cache to make space for new objects
 *
 * it needs to call cache_remove_obj_base before returning
 * which updates some metadata such as n_obj, occupied size, and hash table
 *
 * @param cache
 * @param obj_id
 * @return true if the object is removed, false if the object is not in the
 * cache
 */
static bool ShiftSieve_remove(cache_t *cache, const obj_id_t obj_id) {
  cache_obj_t *obj = hashtable_find_obj_id(cache->hashtable, obj_id);
  if (obj == NULL) {
    return false;
  }

  ShiftSieve_remove_obj(cache, obj);

  return true;
}

static void ShiftSieve_verify(cache_t *cache) {
  ShiftSieve_params_t *params = cache->eviction_params;
  int64_t n_obj = 0, n_byte = 0;
  cache_obj_t *obj = params->q_head;

  while (obj != NULL) {
    assert(hashtable_find_obj_id(cache->hashtable, obj->obj_id) != NULL);
    n_obj++;
    n_byte += obj->obj_size;
    obj = obj->queue.next;
  }

  assert(n_obj == cache->get_n_obj(cache));
  assert(n_byte == cache->get_occupied_byte(cache));
}

#ifdef __cplusplus
}
#endif
