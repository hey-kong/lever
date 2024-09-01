

#include "../../dataStructure/hashtable/hashtable.h"
#include "../../include/libCacheSim/cache.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
  cache_obj_t *q_head;
  cache_obj_t *q_tail;

  cache_obj_t *fast;
  cache_obj_t *slow;
} Lever_params_t;

// ***********************************************************************
// ****                                                               ****
// ****                   function declarations                       ****
// ****                                                               ****
// ***********************************************************************
static void Lever_free(cache_t *cache);
static bool Lever_get(cache_t *cache, const request_t *req);
static cache_obj_t *Lever_find(cache_t *cache, const request_t *req,
                               const bool update_cache);
static cache_obj_t *Lever_insert(cache_t *cache, const request_t *req);
static cache_obj_t *Lever_to_evict(cache_t *cache, const request_t *req);
static void Lever_evict(cache_t *cache, const request_t *req);
static bool Lever_remove(cache_t *cache, const obj_id_t obj_id);

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
cache_t *Lever_init(const common_cache_params_t ccache_params,
                    const char *cache_specific_params) {
  cache_t *cache =
      cache_struct_init("Lever", ccache_params, cache_specific_params);
  cache->cache_init = Lever_init;
  cache->cache_free = Lever_free;
  cache->get = Lever_get;
  cache->find = Lever_find;
  cache->insert = Lever_insert;
  cache->evict = Lever_evict;
  cache->remove = Lever_remove;
  cache->to_evict = Lever_to_evict;

  if (ccache_params.consider_obj_metadata) {
    cache->obj_md_size = 1;
  } else {
    cache->obj_md_size = 0;
  }

  cache->eviction_params = my_malloc(Lever_params_t);
  memset(cache->eviction_params, 0, sizeof(Lever_params_t));
  Lever_params_t *params = (Lever_params_t *)cache->eviction_params;
  params->fast = NULL;
  params->slow = NULL;
  params->q_head = NULL;
  params->q_tail = NULL;

  return cache;
}

/**
 * free resources used by this cache
 *
 * @param cache
 */
static void Lever_free(cache_t *cache) {
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

static bool Lever_get(cache_t *cache, const request_t *req) {
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
static cache_obj_t *Lever_find(cache_t *cache, const request_t *req,
                               const bool update_cache) {
  cache_obj_t *cache_obj = cache_find_base(cache, req, update_cache);
  if (cache_obj != NULL && update_cache) {
    cache_obj->lever.freq = 1;
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
static cache_obj_t *Lever_insert(cache_t *cache, const request_t *req) {
  Lever_params_t *params = cache->eviction_params;
  cache_obj_t *obj = cache_insert_base(cache, req);
  prepend_obj_to_head(&params->q_head, &params->q_tail, obj);
  obj->lever.freq = 0;

  return obj;
}

static cache_obj_t *Lever_to_evict(cache_t *cache, const request_t *req) {
  Lever_params_t *params = cache->eviction_params;
  cache_obj_t *pointer = params->slow;

  if (pointer != NULL && pointer->lever.freq == 0) {
    return pointer;
  }
  return params->q_tail;
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
static void Lever_evict(cache_t *cache, const request_t *req) {
  Lever_params_t *params = cache->eviction_params;

  /* if we have run one full around or first eviction */
  if (params->slow == NULL) params->slow = params->q_tail;
  if (params->fast == NULL) params->fast = params->q_tail;

  for (int i = 0; i < 2; i++) {
    cache_obj_t *obj = params->fast;
    params->fast = params->fast->queue.prev;
    if (obj->lever.freq == 1) {
      obj->lever.freq = 0;
      move_obj_after_mark(&params->q_head, &params->q_tail, &params->slow, obj);
    }
    if (params->fast == NULL) break;
  }

  cache_obj_t *obj = params->slow;
  params->slow = params->slow->queue.prev;
  if (obj->lever.freq == 1) {
    obj->lever.freq = 0;
    /* FIFO demotion */
    cache_obj_t *obj_to_evict = params->q_tail;
    params->q_tail = params->q_tail->queue.prev;
    if (likely(params->q_tail != NULL)) {
      params->q_tail->queue.next = NULL;
    } else {
      /* cache->n_obj has not been updated */
      DEBUG_ASSERT(cache->n_obj == 1);
      params->q_head = NULL;
    }
    cache_evict_base(cache, obj_to_evict, true);
  } else {
    /* quick demotion */
    remove_obj_from_list(&params->q_head, &params->q_tail, obj);
    cache_evict_base(cache, obj, true);
  }
}

static void Lever_remove_obj(cache_t *cache, cache_obj_t *obj_to_remove) {
  DEBUG_ASSERT(obj_to_remove != NULL);
  Lever_params_t *params = cache->eviction_params;
  if (obj_to_remove == params->slow) {
    params->slow = obj_to_remove->queue.prev;
  }
  if (obj_to_remove == params->fast) {
    params->fast = obj_to_remove->queue.prev;
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
static bool Lever_remove(cache_t *cache, const obj_id_t obj_id) {
  cache_obj_t *obj = hashtable_find_obj_id(cache->hashtable, obj_id);
  if (obj == NULL) {
    return false;
  }

  Lever_remove_obj(cache, obj);

  return true;
}

static void Lever_verify(cache_t *cache) {
  Lever_params_t *params = cache->eviction_params;
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
