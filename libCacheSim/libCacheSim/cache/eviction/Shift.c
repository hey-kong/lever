

#include "../../dataStructure/hashtable/hashtable.h"
#include "../../include/libCacheSim/cache.h"
#include "../../include/libCacheSim/evictionAlgo.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
  cache_t *eviction;
  cache_t *retention;

  bool shift;

  request_t *req_local;
} Shift_params_t;

// ***********************************************************************
// ****                                                               ****
// ****                   function declarations                       ****
// ****                                                               ****
// ***********************************************************************
static void Shift_free(cache_t *cache);
static bool Shift_get(cache_t *cache, const request_t *req);
static cache_obj_t *Shift_find(cache_t *cache, const request_t *req,
                               const bool update_cache);
static cache_obj_t *Shift_insert(cache_t *cache, const request_t *req);
static cache_obj_t *Shift_to_evict(cache_t *cache, const request_t *req);
static void Shift_evict(cache_t *cache, const request_t *req);
static bool Shift_remove(cache_t *cache, const obj_id_t obj_id);
static inline int64_t Shift_get_n_obj(const cache_t *cache);
static inline int64_t Shift_get_occupied_byte(const cache_t *cache);

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
cache_t *Shift_init(const common_cache_params_t ccache_params,
                    const char *cache_specific_params) {
  cache_t *cache =
      cache_struct_init("Shift", ccache_params, cache_specific_params);
  cache->cache_init = Shift_init;
  cache->cache_free = Shift_free;
  cache->get = Shift_get;
  cache->find = Shift_find;
  cache->insert = Shift_insert;
  cache->evict = Shift_evict;
  cache->remove = Shift_remove;
  cache->to_evict = Shift_to_evict;
  cache->get_n_obj = Shift_get_n_obj;
  cache->get_occupied_byte = Shift_get_occupied_byte;

  if (ccache_params.consider_obj_metadata) {
    cache->obj_md_size = 1;
  } else {
    cache->obj_md_size = 0;
  }

  cache->eviction_params = my_malloc(Shift_params_t);
  memset(cache->eviction_params, 0, sizeof(Shift_params_t));
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;

  common_cache_params_t ccache_params_eviction = ccache_params;
  params->eviction = FIFO_init(ccache_params_eviction, NULL);

  common_cache_params_t ccache_params_retention = ccache_params;
  params->retention = FIFO_init(ccache_params_retention, NULL);

  params->shift = false;

  params->req_local = new_request();

  return cache;
}

/**
 * free resources used by this cache
 *
 * @param cache
 */
static void Shift_free(cache_t *cache) {
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;
  free_request(params->req_local);
  params->eviction->cache_free(params->eviction);
  params->retention->cache_free(params->retention);
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

static bool Shift_get(cache_t *cache, const request_t *req) {
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
static cache_obj_t *Shift_find(cache_t *cache, const request_t *req,
                               const bool update_cache) {
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;

   // if update cache is false, we only check the eviction and retention caches
  if (!update_cache) {
    cache_obj_t *obj = params->retention->find(params->retention, req, false);
    if (obj != NULL) {
      return obj;
    }
    obj = params->eviction->find(params->eviction, req, false);
    if (obj != NULL) {
      return obj;
    }
    return NULL;
  }

  /* update cache is true from now */
  if (params->eviction != NULL) {
    cache_obj_t *obj = params->eviction->find(params->eviction, req, true);
    if (obj != NULL) {
      if (obj->shift.freq == 0) {
        FIFO_params_t* eviction_params = (FIFO_params_t *)params->eviction->eviction_params;
        move_obj_to_head(&eviction_params->q_head, &eviction_params->q_tail, obj);
      }
      obj->shift.freq += 1;
      return obj;
    }
  }

  if (params->retention != NULL) {
    cache_obj_t *obj = params->retention->find(params->retention, req, true);
    if (obj != NULL) {
      if (obj->shift.freq == 0) {
        FIFO_params_t* retention_params = (FIFO_params_t *)params->retention->eviction_params;
        move_obj_to_head(&retention_params->q_head, &retention_params->q_tail, obj);
      }
      obj->shift.freq += 1;
      return obj;
    }
  }

  return NULL;
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
static cache_obj_t *Shift_insert(cache_t *cache, const request_t *req) {
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;
  cache_obj_t *obj = NULL;
  if (params->shift) {
    obj = params->retention->insert(params->retention, req);
  } else {
    obj = params->eviction->insert(params->eviction, req);
  }
  obj->shift.freq = 0;

  return obj;
}

static cache_obj_t *Shift_to_evict(cache_t *cache, const request_t *req) {
  assert(false);
  return NULL;
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
static void Shift_evict(cache_t *cache, const request_t *req) {
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;
  cache_t *eviction = params->eviction;

  bool has_evicted = false;
  while (!has_evicted && eviction->n_obj > 0) {
    cache_obj_t *obj_to_evict = eviction->to_evict(eviction, req);
    DEBUG_ASSERT(obj_to_evict != NULL);
    copy_cache_obj_to_request(params->req_local, obj_to_evict);
    if (obj_to_evict->shift.freq >= 1) {
      cache_obj_t *new_obj = params->retention->insert(params->retention, params->req_local);
      new_obj->shift.freq /= 2;
    } else {
      has_evicted = true;
    }
    bool removed = eviction->remove(eviction, obj_to_evict->obj_id);
    if (!removed) {
      ERROR("cannot remove obj %ld\n", (long)obj_to_evict->obj_id);
    }
    obj_to_evict = NULL;
    if (eviction->n_obj <= 0) {
      params->eviction = params->retention;
      params->retention = eviction;
      params->shift = false;
    }
  }

  if (params->eviction->n_obj <= cache->get_n_obj(cache) / 10) {
    params->shift = true;
  }
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
static bool Shift_remove(cache_t *cache, const obj_id_t obj_id) {
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;
  bool removed = params->eviction->remove(params->eviction, obj_id);
  if (!removed) {
    removed = params->retention->remove(params->retention, obj_id);
  }

  return removed;
}

static inline int64_t Shift_get_occupied_byte(const cache_t *cache) {
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;
  return params->eviction->get_occupied_byte(params->eviction) +
         params->retention->get_occupied_byte(params->retention);
}

static inline int64_t Shift_get_n_obj(const cache_t *cache) {
  Shift_params_t *params = (Shift_params_t *)cache->eviction_params;
  return params->eviction->get_n_obj(params->eviction) +
         params->retention->get_n_obj(params->retention);
}

#ifdef __cplusplus
}
#endif
