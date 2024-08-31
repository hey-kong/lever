

#include <assert.h>
#include <gmodule.h>

#include "../include/libCacheSim/cacheObj.h"
#include "../include/libCacheSim/macro.h"
#include "../include/libCacheSim/request.h"

/**
 * copy the cache_obj to req_dest
 * @param req_dest
 * @param cache_obj
 */
void copy_cache_obj_to_request(request_t *req_dest,
                               const cache_obj_t *cache_obj) {
  req_dest->obj_id = cache_obj->obj_id;
  req_dest->obj_size = cache_obj->obj_size;
  req_dest->next_access_vtime = cache_obj->misc.next_access_vtime;
  req_dest->valid = true;
}

/**
 * copy the data from request into cache_obj
 * @param cache_obj
 * @param req
 */
void copy_request_to_cache_obj(cache_obj_t *cache_obj, const request_t *req) {
  cache_obj->obj_size = req->obj_size;
#ifdef SUPPORT_TTL
  if (req->ttl != 0)
    cache_obj->exp_time = req->clock_time + req->ttl;
  else
    cache_obj->exp_time = 0;
#endif
  cache_obj->obj_id = req->obj_id;
}

/**
 * create a cache_obj from request
 * @param req
 * @return
 */
cache_obj_t *create_cache_obj_from_request(const request_t *req) {
  cache_obj_t *cache_obj = my_malloc(cache_obj_t);
  memset(cache_obj, 0, sizeof(cache_obj_t));
  if (req != NULL) copy_request_to_cache_obj(cache_obj, req);
  return cache_obj;
}

/** remove the object from the built-in doubly linked list
 *
 * @param head
 * @param tail
 * @param cache_obj
 */
void remove_obj_from_list(cache_obj_t **head, cache_obj_t **tail,
                          cache_obj_t *cache_obj) {
  if (head != NULL && cache_obj == *head) {
    *head = cache_obj->queue.next;
    if (cache_obj->queue.next != NULL) cache_obj->queue.next->queue.prev = NULL;
  }
  if (tail != NULL && cache_obj == *tail) {
    *tail = cache_obj->queue.prev;
    if (cache_obj->queue.prev != NULL) cache_obj->queue.prev->queue.next = NULL;
  }

  if (cache_obj->queue.prev != NULL)
    cache_obj->queue.prev->queue.next = cache_obj->queue.next;

  if (cache_obj->queue.next != NULL)
    cache_obj->queue.next->queue.prev = cache_obj->queue.prev;

  cache_obj->queue.prev = NULL;
  cache_obj->queue.next = NULL;
}

/**
 * move an object to the tail of the doubly linked list
 * @param head
 * @param tail
 * @param cache_obj
 */
void move_obj_to_tail(cache_obj_t **head, cache_obj_t **tail,
                      cache_obj_t *cache_obj) {
  if (*head == *tail) {
    // the list only has one element
    assert(cache_obj == *head);
    assert(cache_obj->queue.next == NULL);
    assert(cache_obj->queue.prev == NULL);
    return;
  }
  if (cache_obj == *head) {
    // change head
    *head = cache_obj->queue.next;
    cache_obj->queue.next->queue.prev = NULL;

    // move to tail
    (*tail)->queue.next = cache_obj;
    cache_obj->queue.next = NULL;
    cache_obj->queue.prev = *tail;
    *tail = cache_obj;
    return;
  }
  if (cache_obj == *tail) {
    return;
  }

  // bridge list_prev and next
  cache_obj->queue.prev->queue.next = cache_obj->queue.next;
  cache_obj->queue.next->queue.prev = cache_obj->queue.prev;

  // handle current tail
  (*tail)->queue.next = cache_obj;

  // handle this moving object
  cache_obj->queue.next = NULL;
  cache_obj->queue.prev = *tail;

  // handle tail
  *tail = cache_obj;
}

/**
 * move an object to the head of the doubly linked list
 * @param head
 * @param tail
 * @param cache_obj
 */
void move_obj_to_head(cache_obj_t **head, cache_obj_t **tail,
                      cache_obj_t *cache_obj) {
  DEBUG_ASSERT(head != NULL);

  if (tail != NULL && *head == *tail) {
    // the list only has one element
    DEBUG_ASSERT(cache_obj == *head);
    DEBUG_ASSERT(cache_obj->queue.next == NULL);
    DEBUG_ASSERT(cache_obj->queue.prev == NULL);
    return;
  }

  if (cache_obj == *head) {
    // already at head
    return;
  }

  if (tail != NULL && cache_obj == *tail) {
    // change tail
    cache_obj->queue.prev->queue.next = cache_obj->queue.next;
    *tail = cache_obj->queue.prev;

    // move to head
    (*head)->queue.prev = cache_obj;
    cache_obj->queue.prev = NULL;
    cache_obj->queue.next = *head;
    *head = cache_obj;
    return;
  }

  // bridge list_prev and next
  cache_obj->queue.prev->queue.next = cache_obj->queue.next;
  cache_obj->queue.next->queue.prev = cache_obj->queue.prev;

  // handle current head
  (*head)->queue.prev = cache_obj;

  // handle this moving object
  cache_obj->queue.prev = NULL;
  cache_obj->queue.next = *head;

  // handle head
  *head = cache_obj;
}

/**
 * move an object to a position after the marked node in the doubly linked list
 * @param head
 * @param tail
 * @param mark
 * @param cache_obj
 */
void move_obj_after_mark(cache_obj_t **head, cache_obj_t **tail,
                         cache_obj_t **mark, cache_obj_t *cache_obj) {
    assert(head != NULL && tail != NULL && mark != NULL && cache_obj != NULL);

    // if cache_obj == mark, the list is not modified.
    if (cache_obj == *mark) {
        return;
    }

    // if the object is the head, update the head to the next node
    if (cache_obj == *head) {
        *head = cache_obj->queue.next;
        if (*head != NULL) {
            (*head)->queue.prev = NULL;
        }
    } else {
        // bridge the previous and next nodes to remove cache_obj from its current position
        cache_obj->queue.prev->queue.next = cache_obj->queue.next;
        if (cache_obj->queue.next != NULL) {
            cache_obj->queue.next->queue.prev = cache_obj->queue.prev;
        } else {
            // if cache_obj was the tail, update the tail to the previous node
            *tail = cache_obj->queue.prev;
        }
    }

    cache_obj->queue.prev = *mark;
    cache_obj->queue.next = (*mark)->queue.next;

    if ((*mark)->queue.next != NULL) {
        (*mark)->queue.next->queue.prev = cache_obj;
    } else {
        *tail = cache_obj;
    }

    (*mark)->queue.next = cache_obj;
}

/**
 * prepend the object to the head of the doubly linked list
 * the object is not in the list, otherwise, use move_obj_to_head
 * @param head
 * @param tail
 * @param cache_obj
 */
void prepend_obj_to_head(cache_obj_t **head, cache_obj_t **tail,
                         cache_obj_t *cache_obj) {
  assert(head != NULL);

  cache_obj->queue.prev = NULL;
  cache_obj->queue.next = *head;

  if (tail != NULL && *tail == NULL) {
    // the list is empty
    DEBUG_ASSERT(*head == NULL);
    *tail = cache_obj;
  }

  if (*head != NULL) {
    // the list has at least one element
    (*head)->queue.prev = cache_obj;
  }

  *head = cache_obj;
}

/**
 * insert the object after the marked node in the doubly linked list
 * @param head
 * @param tail
 * @param mark
 * @param cache_obj
 */
void insert_obj_after_mark(cache_obj_t **head, cache_obj_t **tail,
                           cache_obj_t **mark, cache_obj_t *cache_obj) {
    assert(mark != NULL && cache_obj != NULL);

    cache_obj->queue.prev = *mark;
    cache_obj->queue.next = (*mark)->queue.next;

    if ((*mark)->queue.next != NULL) {
        // There is an element after the mark
        (*mark)->queue.next->queue.prev = cache_obj;
    } else {
        // The mark is the tail of the list
        *tail = cache_obj;
    }

    (*mark)->queue.next = cache_obj;

    // If the mark is the tail, update the tail
    if (*tail == *mark) {
        *tail = cache_obj;
    }
}

/**
 * append the object to the tail of the doubly linked list
 * the object is not in the list, otherwise, use move_obj_to_tail
 * @param head
 * @param tail
 * @param cache_obj
 */
void append_obj_to_tail(cache_obj_t **head, cache_obj_t **tail,
                        cache_obj_t *cache_obj) {

  cache_obj->queue.next = NULL;
  cache_obj->queue.prev = *tail;

  if (head != NULL && *head == NULL) {
    // the list is empty
    DEBUG_ASSERT(*tail == NULL);
    *head = cache_obj;
  }

  if (*tail != NULL) {
    // the list has at least one element
    (*tail)->queue.next = cache_obj;
  }


  *tail = cache_obj;
}