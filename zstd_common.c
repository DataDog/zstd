#ifndef USE_LIBZSTD
/*
 * Copyright (c) 2016-2020, Yann Collet, Facebook, Inc.
 * All rights reserved.
 *
 * This source code is licensed under both the BSD-style license (found in the
 * LICENSE file in the root directory of this source tree) and the GPLv2 (found
 * in the COPYING file in the root directory of this source tree).
 * You may select, at your option, one of the above-listed licenses.
 */



/*-*************************************
*  Dependencies
***************************************/
#define ZSTD_DEPS_NEED_MALLOC
#include "zstd_deps.h"   /* ZSTD_malloc, ZSTD_calloc, ZSTD_free, ZSTD_memset */
#include "error_private.h"
#include "zstd_internal.h"


/*-****************************************
*  Version
******************************************/
unsigned ZSTD_versionNumber(void) { return ZSTD_VERSION_NUMBER; }

const char* ZSTD_versionString(void) { return ZSTD_VERSION_STRING; }


/*-****************************************
*  ZSTD Error Management
******************************************/
#undef ZSTD_isError   /* defined within zstd_internal.h */
/*! ZSTD_isError() :
 *  tells if a return value is an error code
 *  symbol is required for external callers */
unsigned ZSTD_isError(size_t code) { return ERR_isError(code); }

/*! ZSTD_getErrorName() :
 *  provides error code string from function result (useful for debugging) */
const char* ZSTD_getErrorName(size_t code) { return ERR_getErrorName(code); }

/*! ZSTD_getError() :
 *  convert a `size_t` function result into a proper ZSTD_errorCode enum */
ZSTD_ErrorCode ZSTD_getErrorCode(size_t code) { return ERR_getErrorCode(code); }

/*! ZSTD_getErrorString() :
 *  provides error code string from enum */
const char* ZSTD_getErrorString(ZSTD_ErrorCode code) { return ERR_getErrorString(code); }



/*=**************************************************************
*  Custom allocator
****************************************************************/
void* ZSTD_customMalloc(size_t size, ZSTD_customMem customMem)
{
    if (customMem.customAlloc)
        return customMem.customAlloc(customMem.opaque, size);
    return ZSTD_malloc(size);
}

void* ZSTD_customCalloc(size_t size, ZSTD_customMem customMem)
{
    if (customMem.customAlloc) {
        /* calloc implemented as malloc+memset;
         * not as efficient as calloc, but next best guess for custom malloc */
        void* const ptr = customMem.customAlloc(customMem.opaque, size);
        ZSTD_memset(ptr, 0, size);
        return ptr;
    }
    return ZSTD_calloc(1, size);
}

void ZSTD_customFree(void* ptr, ZSTD_customMem customMem)
{
    if (ptr!=NULL) {
        if (customMem.customFree)
            customMem.customFree(customMem.opaque, ptr);
        else
            ZSTD_free(ptr);
    }
}
#endif /* USE_LIBZSTD */
