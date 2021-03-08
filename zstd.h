/*
    zstd - standard compression library
    Header File
    Copyright (C) 2014-2016, Yann Collet.

    BSD 2-Clause License (http://www.opensource.org/licenses/bsd-license.php)

    Redistribution and use in source and binary forms, with or without
    modification, are permitted provided that the following conditions are
    met:
    * Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above
    copyright notice, this list of conditions and the following disclaimer
    in the documentation and/or other materials provided with the
    distribution.
    THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
    "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
    LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
    A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
    OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
    SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
    LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
    DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
    THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
    (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
    OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

    You can contact the author at :
    - zstd source repository : https://github.com/Cyan4973/zstd
*/
#ifndef ZSTD_0_5_X_H
#define ZSTD_0_5_X_H

#if defined (__cplusplus)
extern "C" {
#endif

/*-*************************************
*  Dependencies
***************************************/
#include <stddef.h>   /* size_t */
# include <stdint.h>


/*-***************************************************************
*  Export parameters
*****************************************************************/
/*!
*  ZSTD_0_5_X_DLL_EXPORT :
*  Enable exporting of functions when building a Windows DLL
*/
#if defined(_WIN32) && defined(ZSTD_0_5_X_DLL_EXPORT) && (ZSTD_0_5_X_DLL_EXPORT==1)
#  define ZSTDLIB_API __declspec(dllexport)
#else
#  define ZSTDLIB_API
#endif


/* *************************************
*  Version
***************************************/
#define ZSTD_0_5_X_VERSION_MAJOR    0    /* for breaking interface changes  */
#define ZSTD_0_5_X_VERSION_MINOR    5    /* for new (non-breaking) interface capabilities */
#define ZSTD_0_5_X_VERSION_RELEASE  0    /* for tweaks, bug-fixes, or development */
#define ZSTD_0_5_X_VERSION_NUMBER  (ZSTD_0_5_X_VERSION_MAJOR *100*100 + ZSTD_0_5_X_VERSION_MINOR *100 + ZSTD_0_5_X_VERSION_RELEASE)
ZSTDLIB_API unsigned ZSTD_0_5_X_versionNumber (void);


/* *************************************
*  Simple functions
***************************************/
/*! ZSTD_0_5_X_compress() :
    Compresses `srcSize` bytes from buffer `src` into buffer `dst` of size `dstCapacity`.
    Destination buffer must be already allocated.
    Compression runs faster if `dstCapacity` >=  `ZSTD_0_5_X_compressBound(srcSize)`.
    @return : the number of bytes written into `dst`,
              or an error code if it fails (which can be tested using ZSTD_0_5_X_isError()) */
ZSTDLIB_API size_t ZSTD_0_5_X_compress(   void* dst, size_t dstCapacity,
                              const void* src, size_t srcSize,
                                     int  compressionLevel);

/*! ZSTD_0_5_X_decompress() :
    `compressedSize` : is the _exact_ size of the compressed blob, otherwise decompression will fail.
    `dstCapacity` must be large enough, equal or larger than originalSize.
    @return : the number of bytes decompressed into `dst` (<= `dstCapacity`),
              or an errorCode if it fails (which can be tested using ZSTD_0_5_X_isError()) */
ZSTDLIB_API size_t ZSTD_0_5_X_decompress( void* dst, size_t dstCapacity,
                              const void* src, size_t compressedSize);


/* *************************************
*  Helper functions
***************************************/
ZSTDLIB_API size_t      ZSTD_0_5_X_compressBound(size_t srcSize); /*!< maximum compressed size (worst case scenario) */

/* Error Management */
ZSTDLIB_API unsigned    ZSTD_0_5_X_isError(size_t code);          /*!< tells if a `size_t` function result is an error code */
ZSTDLIB_API const char* ZSTD_0_5_X_getErrorName(size_t code);     /*!< provides readable string for an error code */


/* *************************************
*  Explicit memory management
***************************************/
/** Compression context */
typedef struct ZSTD_0_5_X_CCtx_s ZSTD_0_5_X_CCtx;                       /*< incomplete type */
ZSTDLIB_API ZSTD_0_5_X_CCtx* ZSTD_0_5_X_createCCtx(void);
ZSTDLIB_API size_t     ZSTD_0_5_X_freeCCtx(ZSTD_0_5_X_CCtx* cctx);      /*!< @return : errorCode */

/** ZSTD_0_5_X_compressCCtx() :
    Same as ZSTD_0_5_X_compress(), but requires an already allocated ZSTD_0_5_X_CCtx (see ZSTD_0_5_X_createCCtx()) */
ZSTDLIB_API size_t ZSTD_0_5_X_compressCCtx(ZSTD_0_5_X_CCtx* ctx, void* dst, size_t dstCapacity, const void* src, size_t srcSize, int compressionLevel);

/** Decompression context */
typedef struct ZSTD_0_5_X_DCtx_s ZSTD_0_5_X_DCtx;
ZSTDLIB_API ZSTD_0_5_X_DCtx* ZSTD_0_5_X_createDCtx(void);
ZSTDLIB_API size_t     ZSTD_0_5_X_freeDCtx(ZSTD_0_5_X_DCtx* dctx);      /*!< @return : errorCode */

/** ZSTD_0_5_X_decompressDCtx() :
*   Same as ZSTD_0_5_X_decompress(), but requires an already allocated ZSTD_0_5_X_DCtx (see ZSTD_0_5_X_createDCtx()) */
ZSTDLIB_API size_t ZSTD_0_5_X_decompressDCtx(ZSTD_0_5_X_DCtx* ctx, void* dst, size_t dstCapacity, const void* src, size_t srcSize);


/*-***********************
*  Dictionary API
*************************/
/*! ZSTD_0_5_X_compress_usingDict() :
*   Compression using a pre-defined Dictionary content (see dictBuilder).
*   Note : dict can be NULL, in which case, it's equivalent to ZSTD_0_5_X_compressCCtx() */
ZSTDLIB_API size_t ZSTD_0_5_X_compress_usingDict(ZSTD_0_5_X_CCtx* ctx,
                                           void* dst, size_t dstCapacity,
                                     const void* src, size_t srcSize,
                                     const void* dict,size_t dictSize,
                                           int compressionLevel);

/*! ZSTD_0_5_X_decompress_usingDict() :
*   Decompression using a pre-defined Dictionary content (see dictBuilder).
*   Dictionary must be identical to the one used during compression, otherwise regenerated data will be corrupted.
*   Note : dict can be NULL, in which case, it's equivalent to ZSTD_0_5_X_decompressDCtx() */
ZSTDLIB_API size_t ZSTD_0_5_X_decompress_usingDict(ZSTD_0_5_X_DCtx* dctx,
                                             void* dst, size_t dstCapacity,
                                       const void* src, size_t srcSize,
                                       const void* dict,size_t dictSize);

/* **************************************
*  Streaming functions (direct mode)
****************************************/
ZSTDLIB_API size_t ZSTD_0_5_X_compressBegin(ZSTD_0_5_X_CCtx* cctx, int compressionLevel);
ZSTDLIB_API size_t ZSTD_0_5_X_compressBegin_usingDict(ZSTD_0_5_X_CCtx* cctx, const void* dict,size_t dictSize, int compressionLevel);
ZSTDLIB_API size_t ZSTD_0_5_X_compressContinue(ZSTD_0_5_X_CCtx* cctx, void* dst, size_t dstCapacity, const void* src, size_t srcSize);
ZSTDLIB_API size_t ZSTD_0_5_X_compressEnd(ZSTD_0_5_X_CCtx* cctx, void* dst, size_t dstCapacity);


#if defined (__cplusplus)
}
#endif

#endif  /* ZSTD_0_5_X_H */
